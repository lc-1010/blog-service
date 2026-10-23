package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"blog-server/global"
	"blog-server/internal/platform"
	"blog-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	sessionCookieName = "blog_admin_session"
	sessionDuration   = 8 * time.Hour
)

type Handler struct {
	username string
	password string
	secret   []byte
}

func (h Handler) AuditMutations() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			c.Next()
			if c.FullPath() == "" || global.DBEngine == nil {
				return
			}
			action := strings.TrimPrefix(c.FullPath(), "/api/v1/")
			if err := platform.RecordAudit(
				global.DBEngine.WithContext(c.Request.Context()),
				currentUsername(c),
				action,
				c.FullPath(),
				c.Request.Method,
				c.ClientIP(),
				c.Writer.Status(),
			); err != nil && global.Logger != nil {
				global.Logger.Errorf(c, "record admin operation audit err: %v", err)
			}
		default:
			c.Next()
		}
	}
}

func NewHandler() Handler {
	return Handler{
		username: os.Getenv("BLOG_ADMIN_USERNAME"),
		password: os.Getenv("BLOG_ADMIN_PASSWORD"),
		secret:   []byte(os.Getenv("BLOG_ADMIN_SESSION_SECRET")),
	}
}

func (h Handler) RegisterRoutes(router *gin.Engine) {
	admin := router.Group("/api/v1/admin")
	admin.POST("/login", h.Login)
	admin.POST("/logout", h.Logout)
	admin.GET("/me", h.AuthRequired(), h.Me)
	admin.GET("/dashboard", h.AuthRequired(), h.Dashboard)
	h.registerPlatformRoutes(router)
}

func (h Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(h.secret) < 32 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "admin session secret is not configured"})
			c.Abort()
			return
		}

		cookie, err := c.Cookie(sessionCookieName)
		if err != nil || !h.validSession(cookie) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		payload, err := base64.RawURLEncoding.DecodeString(strings.Split(cookie, ".")[0])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		fields := strings.Split(string(payload), "\n")
		if len(fields) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		username := fields[0]
		if h.username != "" && username == h.username {
			c.Set("admin_username", username)
			c.Set("admin_user_id", uint32(0))
			c.Set("admin_role_name", "超级管理员")
			c.Set("admin_permissions", []string{"*"})
			c.Next()
			return
		}
		if global.DBEngine == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
			c.Abort()
			return
		}
		var user platform.User
		err = global.DBEngine.WithContext(c.Request.Context()).
			Where("username = ? AND state = ? AND is_del = ?", username, 1, 0).
			First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			} else {
				platformRequestError(c, err)
			}
			c.Abort()
			return
		}
		var role platform.Role
		if err := global.DBEngine.WithContext(c.Request.Context()).First(&role, user.RoleID).Error; err != nil {
			platformRequestError(c, err)
			c.Abort()
			return
		}
		c.Set("admin_username", user.Username)
		c.Set("admin_user_id", user.ID)
		c.Set("admin_display_name", user.DisplayName)
		c.Set("admin_role_name", role.Name)
		c.Set("admin_permissions", role.PermissionList())
		c.Next()
	}
}

func (h Handler) Login(c *gin.Context) {
	if len(h.secret) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "admin session secret is not configured"})
		return
	}

	var request struct {
		Username string `json:"username" binding:"required,max=100"`
		Password string `json:"password" binding:"required,max=256"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid username or password"})
		return
	}
	isBootstrapUsername := h.username != "" &&
		subtle.ConstantTimeCompare([]byte(request.Username), []byte(h.username)) == 1
	isBootstrapAdmin := isBootstrapUsername && h.password != "" &&
		subtle.ConstantTimeCompare([]byte(request.Password), []byte(h.password)) == 1
	if isBootstrapUsername && !isBootstrapAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid username or password"})
		return
	}
	var user platform.User
	if !isBootstrapAdmin {
		if global.DBEngine == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
			return
		}
		err := global.DBEngine.WithContext(c.Request.Context()).
			Where("username = ? AND state = ? AND is_del = ?", request.Username, 1, 0).
			First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && !platform.VerifyPassword(user.PasswordHash, request.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid username or password"})
			return
		}
		if err != nil {
			platformRequestError(c, err)
			return
		}
	}
	if !isBootstrapAdmin && user.ID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid username or password"})
		return
	}

	expiresAt := time.Now().Add(sessionDuration).Unix()
	username := request.Username
	payload := base64.RawURLEncoding.EncodeToString([]byte(username + "\n" + strconv.FormatInt(expiresAt, 10)))
	token := payload + "." + h.sign(payload)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureRequest(c),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
		Expires:  time.Unix(expiresAt, 0),
	})
	if global.DBEngine != nil {
		if err := platform.RecordAudit(global.DBEngine.WithContext(c.Request.Context()), username, "auth.login", "admin session", http.MethodPost, c.ClientIP(), http.StatusOK); err != nil && global.Logger != nil {
			global.Logger.Errorf(c, "record admin login audit err: %v", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"username": username})
}

func (h Handler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureRequest(c),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h Handler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id":           currentUserID(c),
		"username":     currentUsername(c),
		"display_name": contextString(c, "admin_display_name"),
		"role":         contextString(c, "admin_role_name"),
		"permissions":  contextPermissions(c),
	})
}

func (h Handler) Dashboard(c *gin.Context) {
	svc := service.New(c.Request.Context())
	stats, err := svc.GetDashboardStats()
	if err != nil {
		if global.Logger != nil {
			global.Logger.Errorf(c, "GetDashboardStats err: %v", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load dashboard"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h Handler) configured() bool {
	return h.username != "" && h.password != "" && len(h.secret) >= 32
}

func (h Handler) validSession(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(h.sign(parts[0]))) {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	fields := strings.Split(string(payload), "\n")
	if len(fields) != 2 || fields[0] == "" {
		return false
	}
	expiresAt, err := strconv.ParseInt(fields[1], 10, 64)
	return err == nil && time.Now().Unix() < expiresAt
}

func (h Handler) sign(payload string) string {
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func secureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

func contextString(c *gin.Context, key string) string {
	value, _ := c.Get(key)
	result, _ := value.(string)
	return result
}

func contextPermissions(c *gin.Context) []string {
	value, _ := c.Get("admin_permissions")
	result, _ := value.([]string)
	return result
}

func platformRequestError(c *gin.Context, err error) {
	if global.Logger != nil {
		global.Logger.Errorf(c, "load admin principal err: %v", err)
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load admin permissions"})
}
