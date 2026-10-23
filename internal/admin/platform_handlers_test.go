package admin_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog-server/global"
	"blog-server/internal/admin"
	"blog-server/internal/membership"
	"blog-server/internal/model"
	"blog-server/internal/platform"
	adminrouter "blog-server/internal/routers/admin"
	"blog-server/pkg/setting"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const adminSessionCookieName = "blog_admin_session"

func TestPlatformUsersSettingsSitesAndAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("BLOG_ADMIN_USERNAME", "admin-test")
	t.Setenv("BLOG_ADMIN_PASSWORD", "correct-horse-battery")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "a-test-secret-that-is-at-least-32-bytes-long")

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:platform-admin-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.BlogTag{}, &model.BlogArticle{}, &model.BlogArticleTag{},
		&membership.Member{}, &membership.Profile{}, &membership.Session{},
		&platform.Role{}, &platform.User{}, &platform.Setting{}, &platform.Site{}, &platform.AuditLog{},
	); err != nil {
		t.Fatalf("migrate platform models: %v", err)
	}
	if err := platform.EnsureDefaultRoles(db); err != nil {
		t.Fatalf("create default roles: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	oldDB, oldAppSetting := global.DBEngine, global.AppSetting
	global.DBEngine = db
	global.AppSetting = &setting.AppSettingS{DefaultPageSize: 10, MaxPageSize: 100}
	t.Cleanup(func() {
		global.DBEngine, global.AppSetting = oldDB, oldAppSetting
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close SQL database: %v", err)
		}
	})

	var editorRole platform.Role
	if err := db.Where("name = ?", "内容编辑").First(&editorRole).Error; err != nil {
		t.Fatalf("load editor role: %v", err)
	}
	router := adminrouter.New()
	rootCookie := loginCookieWithCredentials(t, router, "admin-test", "correct-horse-battery")

	createdUser := performPlatformRequest(router, rootCookie, http.MethodPost, "/api/v1/admin/platform/users", `{"username":"writer01","display_name":"文章编辑","password":"correct-long-password","role_id":`+fmt.Sprint(editorRole.ID)+`}`)
	if createdUser.Code != http.StatusCreated {
		t.Fatalf("create user status = %d, body = %s", createdUser.Code, createdUser.Body.String())
	}
	var persistedUser platform.User
	if err := db.Where("username = ?", "writer01").First(&persistedUser).Error; err != nil {
		t.Fatalf("load created user: %v", err)
	}
	if !platform.VerifyPassword(persistedUser.PasswordHash, "correct-long-password") {
		t.Fatal("created user password hash does not match the submitted password")
	}
	writerCookie := loginCookieWithCredentials(t, router, "writer01", "correct-long-password")
	me := performPlatformRequest(router, writerCookie, http.MethodGet, "/api/v1/admin/me", "")
	if me.Code != http.StatusOK {
		t.Fatalf("managed user identity status = %d, body = %s", me.Code, me.Body.String())
	}
	var principal struct {
		Permissions []string `json:"permissions"`
		Role        string   `json:"role"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &principal); err != nil {
		t.Fatalf("decode managed user identity: %v", err)
	}
	if principal.Role != "内容编辑" || len(principal.Permissions) != 1 || principal.Permissions[0] != platform.PermissionContentManage {
		t.Fatalf("unexpected managed user identity: %+v", principal)
	}
	denied := performPlatformRequest(router, writerCookie, http.MethodGet, "/api/v1/admin/platform/users", "")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("users API status = %d, want %d; body = %s", denied.Code, http.StatusForbidden, denied.Body.String())
	}
	contentAccess := performPlatformRequest(router, writerCookie, http.MethodGet, "/api/v1/articles?page=1&page_size=10", "")
	if contentAccess.Code != http.StatusOK {
		t.Fatalf("content permission status = %d, body = %s", contentAccess.Code, contentAccess.Body.String())
	}

	memberPasswordHash, err := platform.HashPassword("member-password-for-test")
	if err != nil {
		t.Fatalf("hash member password: %v", err)
	}
	member := membership.Member{
		Username: "member01", DisplayName: "普通会员",
		PasswordHash: memberPasswordHash, MemberType: "member", State: 1,
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member fixture: %v", err)
	}
	memberList := performPlatformRequest(router, rootCookie, http.MethodGet, "/api/v1/admin/platform/members?page=1&page_size=10", "")
	if memberList.Code != http.StatusOK || bytes.Contains(memberList.Body.Bytes(), []byte(memberPasswordHash)) {
		t.Fatalf("member list status/body = %d/%s", memberList.Code, memberList.Body.String())
	}
	memberUpdate := performPlatformRequest(router, rootCookie, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/platform/members/%d", member.ID),
		`{"member_type":"author","state":1}`)
	if memberUpdate.Code != http.StatusOK {
		t.Fatalf("member role update status/body = %d/%s", memberUpdate.Code, memberUpdate.Body.String())
	}
	var promoted membership.Member
	if err := db.First(&promoted, member.ID).Error; err != nil || promoted.MemberType != "author" {
		t.Fatalf("member promotion result = %+v, err = %v", promoted, err)
	}

	settings := performPlatformRequest(router, rootCookie, http.MethodPut, "/api/v1/admin/platform/settings", `{"site_name":"纸间空间","maintenance_mode":"false"}`)
	if settings.Code != http.StatusOK {
		t.Fatalf("update settings status = %d, body = %s", settings.Code, settings.Body.String())
	}
	invalidTimezone := performPlatformRequest(router, rootCookie, http.MethodPut, "/api/v1/admin/platform/settings", `{"timezone":"Invalid/Zone"}`)
	if invalidTimezone.Code != http.StatusBadRequest {
		t.Fatalf("invalid timezone status = %d, want %d", invalidTimezone.Code, http.StatusBadRequest)
	}
	settings = performPlatformRequest(router, rootCookie, http.MethodGet, "/api/v1/admin/platform/settings", "")
	if settings.Code != http.StatusOK || !bytes.Contains(settings.Body.Bytes(), []byte("纸间空间")) {
		t.Fatalf("read settings status/body = %d/%s", settings.Code, settings.Body.String())
	}

	site := performPlatformRequest(router, rootCookie, http.MethodPost, "/api/v1/admin/platform/sites", `{"name":"纸间博客","domain":"blog.example.com","description":"主站","state":1}`)
	if site.Code != http.StatusCreated {
		t.Fatalf("create site status = %d, body = %s", site.Code, site.Body.String())
	}
	invalidSite := performPlatformRequest(router, rootCookie, http.MethodPost, "/api/v1/admin/platform/sites", `{"name":"无效站点","domain":"blog.example.com/path","description":"","state":1}`)
	if invalidSite.Code != http.StatusBadRequest {
		t.Fatalf("invalid site domain status = %d, want %d; body = %s", invalidSite.Code, http.StatusBadRequest, invalidSite.Body.String())
	}
	site = performPlatformRequest(router, rootCookie, http.MethodGet, "/api/v1/admin/platform/sites", "")
	if site.Code != http.StatusOK || !bytes.Contains(site.Body.Bytes(), []byte("blog.example.com")) {
		t.Fatalf("list sites status/body = %d/%s", site.Code, site.Body.String())
	}

	logs := performPlatformRequest(router, rootCookie, http.MethodGet, "/api/v1/admin/platform/audit-logs?page=1&page_size=50", "")
	if logs.Code != http.StatusOK || !bytes.Contains(logs.Body.Bytes(), []byte("admin/platform/users")) ||
		!bytes.Contains(logs.Body.Bytes(), []byte("admin/platform/settings")) ||
		!bytes.Contains(logs.Body.Bytes(), []byte(`"status":400`)) {
		t.Fatalf("audit log response = %d, body = %s", logs.Code, logs.Body.String())
	}
	var userCreateAudits int64
	if err := db.Model(&platform.AuditLog{}).Where("action = ?", "admin/platform/users").Count(&userCreateAudits).Error; err != nil {
		t.Fatalf("count user management audit entries: %v", err)
	}
	if userCreateAudits != 1 {
		t.Fatalf("user management audit entries = %d, want exactly one", userCreateAudits)
	}
}

func TestRequirePermissionDeniesMissingPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", (admin.Handler{}).RequirePermission("content.manage"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("permission status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func loginCookieWithCredentials(t *testing.T, router http.Handler, username, password string) *http.Cookie {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("encode login request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == adminSessionCookieName {
			return cookie
		}
	}
	t.Fatal("login response did not set session cookie")
	return nil
}

func performPlatformRequest(router http.Handler, cookie *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(cookie)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
