package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/internal/platform"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func configureTestCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("BLOG_ADMIN_USERNAME", "admin-test")
	t.Setenv("BLOG_ADMIN_PASSWORD", "correct-horse-battery")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "a-test-secret-that-is-at-least-32-bytes-long")
}

func TestLoginAndSessionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureTestCredentials(t)

	handler := NewHandler()
	router := gin.New()
	handler.RegisterRoutes(router)
	router.GET("/private", handler.AuthRequired(), func(c *gin.Context) {
		c.String(http.StatusOK, "private")
	})

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"username":"admin-test","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid credentials status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"username":"admin-test","password":"correct-horse-battery"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login returned %d cookies, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge <= 0 {
		t.Fatalf("unexpected session cookie attributes: %+v", cookie)
	}

	request = httptest.NewRequest(http.MethodGet, "/private", nil)
	request.AddCookie(cookie)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "private" {
		t.Fatalf("authenticated request status/body = %d/%q", recorder.Code, recorder.Body.String())
	}

	cookie.Value += "tampered"
	request = httptest.NewRequest(http.MethodGet, "/private", nil)
	request.AddCookie(cookie)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("tampered session status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthenticationRequiresCompleteConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("BLOG_ADMIN_USERNAME", "admin-test")
	t.Setenv("BLOG_ADMIN_PASSWORD", "")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "too-short")

	handler := NewHandler()
	router := gin.New()
	handler.RegisterRoutes(router)
	router.GET("/private", handler.AuthRequired(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"username":"admin-test","password":"anything"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured login status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	request = httptest.NewRequest(http.MethodGet, "/private", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured management status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestAdminDashboardCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configureTestCredentials(t)
	db, err := gorm.Open(sqlite.Open("file:admin-dashboard-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.BlogTag{}, &model.BlogArticle{}, &model.BlogArticleTag{},
		&platform.Role{}, &platform.User{}, &platform.Setting{}, &platform.Site{}, &platform.AuditLog{},
	); err != nil {
		t.Fatalf("migrate blog and platform models: %v", err)
	}
	if err := platform.EnsureDefaultRoles(db); err != nil {
		t.Fatalf("create default roles: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite database: %v", err)
		}
	})
	previousDB := global.DBEngine
	global.DBEngine = db
	t.Cleanup(func() { global.DBEngine = previousDB })

	tag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: "golang", State: 1}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag fixture: %v", err)
	}
	articles := []model.BlogArticle{
		{Model: &model.Model{CreatedBy: "tester"}, Title: "Published", State: 1},
		{Model: &model.Model{CreatedBy: "tester"}, Title: "Draft", State: 0},
		{Model: &model.Model{CreatedBy: "tester", IsDel: 1}, Title: "Deleted", State: 1},
	}
	for i := range articles {
		if err := db.Create(&articles[i]).Error; err != nil {
			t.Fatalf("create article fixture: %v", err)
		}
	}

	router := gin.New()
	NewHandler().RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	request.AddCookie(loginCookie(t, router))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var stats struct {
		Articles  int64 `json:"articles"`
		Published int64 `json:"published"`
		Drafts    int64 `json:"drafts"`
		Tags      int64 `json:"tags"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode dashboard response: %v", err)
	}
	if stats.Articles != 2 || stats.Published != 1 || stats.Drafts != 1 || stats.Tags != 1 {
		t.Fatalf("unexpected dashboard stats: %+v", stats)
	}
}

func loginCookie(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"username":"admin-test","password":"correct-horse-battery"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	t.Fatal("login response did not set a session cookie")
	return nil
}
