package membership

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/internal/platform"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	sessionCookieName = "blog_member_session"
	sessionDuration   = 30 * 24 * time.Hour
)

type Member struct {
	model.Model
	Username      string `json:"username" gorm:"size:100;uniqueIndex"`
	DisplayName   string `json:"display_name" gorm:"size:100"`
	PasswordHash  string `json:"-" gorm:"size:100"`
	Email         string `json:"email" gorm:"size:255"`
	EmailVerified uint8  `json:"email_verified" gorm:"default:0"`
	MemberType    string `json:"member_type" gorm:"size:20;index"`
	State         uint8  `json:"-" gorm:"default:1;index"`
}

func (Member) TableName() string { return "blog_members" }

type Profile struct {
	MemberID  uint32 `json:"member_id" gorm:"primaryKey"`
	Bio       string `json:"bio" gorm:"type:text"`
	AvatarURL string `json:"avatar_url" gorm:"size:500"`
}

func (Profile) TableName() string { return "blog_member_profiles" }

type Session struct {
	ID        uint32     `json:"-" gorm:"primaryKey"`
	MemberID  uint32     `json:"-" gorm:"index"`
	TokenHash string     `json:"-" gorm:"size:64;uniqueIndex"`
	ExpiresAt time.Time  `json:"-"`
	RevokedAt *time.Time `json:"-"`
	CreatedAt time.Time  `json:""`
}

func (Session) TableName() string { return "blog_member_sessions" }

type EmailVerificationCode struct {
	ID        uint32    `json:"-" gorm:"primaryKey"`
	MemberID  uint32    `json:"-" gorm:"index"`
	Code      string    `json:"-" gorm:"size:6"`
	ExpiresAt time.Time `json:"-"`
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}

func (EmailVerificationCode) TableName() string { return "blog_email_verification_codes" }

const verificationCodeExpiry = 10 * time.Minute

type Handler struct {
	db func() *gorm.DB
}

func NewHandler() Handler {
	return Handler{db: func() *gorm.DB { return global.DBEngine }}
}

func (h Handler) RegisterRoutes(router *gin.Engine) {
	members := router.Group("/api/v1/public")
	members.POST("/members/register", h.Register)
	members.POST("/members/login", h.Login)
	members.POST("/members/logout", h.Logout)
	members.GET("/members/me", h.AuthRequired(), h.Me)
	members.PATCH("/members/me/profile", h.AuthRequired(), h.UpdateProfile)
	members.PATCH("/members/me/password", h.AuthRequired(), h.ChangePassword)
	members.GET("/members/:username", h.PublicProfile)
	// Email verification routes
	members.POST("/members/send-verification", h.AuthRequired(), h.SendVerification)
	members.POST("/members/verify-email", h.AuthRequired(), h.VerifyEmail)
}

func (h Handler) Register(c *gin.Context) {
	db, ok := h.database(c)
	if !ok {
		return
	}
	var registrationSetting modelSetting
	err := db.Table("blog_platform_settings").
		Select("value").
		Where("setting_key = ? AND is_del = ?", "allow_registration", 0).
		Take(&registrationSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		memberError(c, err)
		return
	}
	if registrationSetting.Value != "true" {
		c.JSON(http.StatusForbidden, gin.H{"message": "member registration is disabled"})
		return
	}

	var input struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		Email       string `json:"email"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil ||
		!validUsername(input.Username) ||
		strings.TrimSpace(input.DisplayName) == "" ||
		len([]rune(strings.TrimSpace(input.DisplayName))) > 100 ||
		len(input.Password) < 12 || len(input.Password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username, display_name, and a 12-72 byte password are required"})
		return
	}

	hash, err := platform.HashPassword(input.Password)
	if err != nil {
		memberError(c, err)
		return
	}
	member := Member{
		Username:     strings.TrimSpace(input.Username),
		DisplayName:  strings.TrimSpace(input.DisplayName),
		PasswordHash: hash,
		Email:        strings.TrimSpace(input.Email),
		MemberType:   "member",
		State:        1,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return tx.Create(&Profile{MemberID: member.ID}).Error
	})
	if err != nil {
		if duplicateMember(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "username already exists"})
			return
		}
		memberError(c, err)
		return
	}
	token, expiresAt, err := h.createSession(db, member.ID)
	if err != nil {
		memberError(c, err)
		return
	}
	setMemberCookie(c, token, expiresAt)
	c.JSON(http.StatusCreated, memberResponse{ID: member.ID, Username: member.Username, DisplayName: member.DisplayName, MemberType: member.MemberType})
}

func (h Handler) Login(c *gin.Context) {
	db, ok := h.database(c)
	if !ok {
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Username) == "" || len(input.Password) < 1 || len(input.Password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username and password are required"})
		return
	}
	var member Member
	err := db.Where("username = ? AND state = ? AND is_del = ?", strings.TrimSpace(input.Username), 1, 0).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && !platform.VerifyPassword(member.PasswordHash, input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid username or password"})
		return
	}
	if err != nil {
		memberError(c, err)
		return
	}
	token, expiresAt, err := h.createSession(db, member.ID)
	if err != nil {
		memberError(c, err)
		return
	}
	setMemberCookie(c, token, expiresAt)
	c.JSON(http.StatusOK, memberResponse{ID: member.ID, Username: member.Username, DisplayName: member.DisplayName, MemberType: member.MemberType})
}

func (h Handler) Logout(c *gin.Context) {
	db, ok := h.database(c)
	if !ok {
		return
	}
	if cookie, err := c.Cookie(sessionCookieName); err == nil {
		now := time.Now()
		if err := db.Model(&Session{}).
			Where("token_hash = ? AND revoked_at IS NULL", tokenDigest(cookie)).
			Update("revoked_at", now).Error; err != nil {
			memberError(c, err)
			return
		}
	}
	clearMemberCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := h.database(c)
		if !ok {
			return
		}
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil || cookie == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		var session Session
		err = db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenDigest(cookie), time.Now()).
			First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		if err != nil {
			memberError(c, err)
			c.Abort()
			return
		}
		var member Member
		err = db.Where("id = ? AND state = ? AND is_del = ?", session.MemberID, 1, 0).First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
			c.Abort()
			return
		}
		if err != nil {
			memberError(c, err)
			c.Abort()
			return
		}
		c.Set("member_id", member.ID)
		c.Set("member_username", member.Username)
		c.Set("member_display_name", member.DisplayName)
		c.Set("member_type", member.MemberType)
		c.Next()
	}
}

func (h Handler) Me(c *gin.Context) {
	memberID := currentMemberID(c)
	db, ok := h.database(c)
	if !ok {
		return
	}
	var profile Profile
	if err := db.Where("member_id = ?", memberID).First(&profile).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		memberError(c, err)
		return
	}
	c.JSON(http.StatusOK, profileResponse{
		ID: memberID, Username: contextValue(c, "member_username"),
		DisplayName: contextValue(c, "member_display_name"),
		MemberType:  contextValue(c, "member_type"), Bio: profile.Bio, AvatarURL: profile.AvatarURL,
	})
}

func (h Handler) UpdateProfile(c *gin.Context) {
	var input struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
		AvatarURL   string `json:"avatar_url"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil ||
		strings.TrimSpace(input.DisplayName) == "" ||
		len([]rune(strings.TrimSpace(input.DisplayName))) > 100 ||
		len([]rune(input.Bio)) > 2000 ||
		len(input.AvatarURL) > 500 ||
		!validHTTPURL(input.AvatarURL) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid display_name, bio, or avatar_url"})
		return
	}
	db, ok := h.database(c)
	if !ok {
		return
	}
	memberID := currentMemberID(c)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Member{}).Where("id = ?", memberID).Update("display_name", strings.TrimSpace(input.DisplayName)).Error; err != nil {
			return err
		}
		profile := Profile{MemberID: memberID, Bio: strings.TrimSpace(input.Bio), AvatarURL: strings.TrimSpace(input.AvatarURL)}
		return tx.Where(Profile{MemberID: memberID}).Assign(profile).FirstOrCreate(&profile).Error
	})
	if err != nil {
		memberError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h Handler) ChangePassword(c *gin.Context) {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil ||
		len(input.CurrentPassword) < 1 || len(input.CurrentPassword) > 72 ||
		len(input.NewPassword) < 12 || len(input.NewPassword) > 72 ||
		input.CurrentPassword == input.NewPassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "current password and a different 12-72 byte new password are required"})
		return
	}
	db, ok := h.database(c)
	if !ok {
		return
	}
	memberID := currentMemberID(c)
	var member Member
	if err := db.Where("id = ? AND state = ? AND is_del = ?", memberID, 1, 0).First(&member).Error; err != nil {
		memberError(c, err)
		return
	}
	if !platform.VerifyPassword(member.PasswordHash, input.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "current password is incorrect"})
		return
	}
	hash, err := platform.HashPassword(input.NewPassword)
	if err != nil {
		memberError(c, err)
		return
	}
	now := time.Now()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Member{}).Where("id = ?", memberID).Update("password_hash", hash).Error; err != nil {
			return err
		}
		return tx.Model(&Session{}).
			Where("member_id = ? AND revoked_at IS NULL", memberID).
			Update("revoked_at", now).Error
	})
	if err != nil {
		memberError(c, err)
		return
	}
	clearMemberCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "password changed; sign in again"})
}

func (h Handler) PublicProfile(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if !validUsername(username) {
		c.JSON(http.StatusNotFound, gin.H{"message": "member not found"})
		return
	}
	db, ok := h.database(c)
	if !ok {
		return
	}
	var member Member
	if err := db.Where("username = ? AND state = ? AND is_del = ?", username, 1, 0).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "member not found"})
		} else {
			memberError(c, err)
		}
		return
	}
	var profile Profile
	if err := db.Where("member_id = ?", member.ID).First(&profile).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		memberError(c, err)
		return
	}
	c.JSON(http.StatusOK, profileResponse{
		ID: member.ID, Username: member.Username, DisplayName: member.DisplayName,
		MemberType: member.MemberType, Bio: profile.Bio, AvatarURL: profile.AvatarURL,
	})
}

func (h Handler) database(c *gin.Context) (*gorm.DB, bool) {
	db := h.db()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return nil, false
	}
	return db.WithContext(c.Request.Context()), true
}

func (h Handler) createSession(db *gorm.DB, memberID uint32) (string, time.Time, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	expiresAt := time.Now().Add(sessionDuration)
	session := Session{MemberID: memberID, TokenHash: tokenDigest(token), ExpiresAt: expiresAt, CreatedAt: time.Now()}
	if err := db.Create(&session).Error; err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

type modelSetting struct {
	Value string
}

type memberResponse struct {
	ID          uint32 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	MemberType  string `json:"member_type"`
}

type profileResponse struct {
	ID          uint32 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	MemberType  string `json:"member_type"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
}

func setMemberCookie(c *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: secureRequest(c), SameSite: http.SameSiteStrictMode,
		MaxAge: int(time.Until(expiresAt).Seconds()), Expires: expiresAt,
	})
}

func clearMemberCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: secureRequest(c), SameSite: http.SameSiteStrictMode,
		MaxAge: -1, Expires: time.Unix(0, 0),
	})
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func validUsername(value string) bool {
	if len(value) < 3 || len(value) > 100 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' || char == '.' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func validHTTPURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" &&
		(parsed.Scheme == "https" || parsed.Scheme == "http") &&
		parsed.User == nil
}

func currentMemberID(c *gin.Context) uint32 {
	value, _ := c.Get("member_id")
	memberID, _ := value.(uint32)
	return memberID
}

func contextValue(c *gin.Context, key string) string {
	value, _ := c.Get(key)
	result, _ := value.(string)
	return result
}

func secureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

func (h Handler) SendVerification(c *gin.Context) {
	db, ok := h.database(c)
	if !ok {
		return
	}
	memberID := currentMemberID(c)

	var member Member
	if err := db.Where("id = ? AND is_del = ?", memberID, 0).First(&member).Error; err != nil {
		memberError(c, err)
		return
	}
	if member.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "no email address on file"})
		return
	}

	// Clean up expired or used codes for this member
	now := time.Now()
	db.Where("member_id = ?", memberID).Delete(EmailVerificationCode{})

	code := generateVerificationCode()
	expiry := now.Add(verificationCodeExpiry)

	verification := EmailVerificationCode{
		MemberID:  memberID,
		Code:      code,
		ExpiresAt: expiry,
		CreatedAt: now,
	}
	if err := db.Create(&verification).Error; err != nil {
		memberError(c, err)
		return
	}

	// In production, send an actual email here.
	// For now, log the verification code so it can be used in tests.
	if global.Logger != nil {
		global.Logger.Infof(c, "verification code for %s: %s (expires in %v)", member.Email, code, verificationCodeExpiry)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("verification code sent to %s, valid for %v", member.Email, verificationCodeExpiry)})
}

func (h Handler) VerifyEmail(c *gin.Context) {
	db, ok := h.database(c)
	if !ok {
		return
	}
	memberID := currentMemberID(c)

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request payload"})
		return
	}

	var verification EmailVerificationCode
	if err := db.Where("member_id = ? AND code = ? AND expires_at > ? AND used_at IS NULL",
		memberID, input.Code, time.Now()).First(&verification).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid or expired verification code"})
		return
	}

	// Mark the code as used
	usedAt := time.Now()
	db.Model(&verification).Update("used_at", usedAt)

	// Mark the member's email as verified
	db.Model(&Member{}).Where("id = ?", memberID).Update("email_verified", 1)

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

// generateVerificationCode returns a random 6-digit numeric string.
func generateVerificationCode() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use a simple counter-based approach
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	code := fmt.Sprintf("%06d", int(b[0])%100*10000+int(b[1])%100*100+int(b[2])%100)
	return code
}

func memberError(c *gin.Context, err error) {
	if global.Logger != nil {
		global.Logger.Errorf(c, "member request failed: %v", err)
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "member request failed"})
}

func duplicateMember(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint")
}
