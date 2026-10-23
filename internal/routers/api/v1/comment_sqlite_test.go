package v1_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"blog-server/global"
	"blog-server/internal/membership"
	"blog-server/internal/model"
	"blog-server/internal/platform"
	adminrouter "blog-server/internal/routers/admin"
	publicrouter "blog-server/internal/routers/public"
	"blog-server/pkg/setting"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommentFlowRequiresApproval(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("BLOG_ADMIN_USERNAME", "admin")
	t.Setenv("BLOG_ADMIN_PASSWORD", "correct-horse-battery")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "a-secret-that-is-at-least-32-bytes-long")

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:comment-flow-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.BlogTag{},
		&model.BlogArticle{},
		&model.BlogArticleTag{},
		&membership.Member{},
		&membership.Profile{},
		&membership.Session{},
		&model.BlogComment{},
		&platform.Role{},
		&platform.User{},
		&platform.Setting{},
		&platform.Site{},
		&platform.AuditLog{},
	); err != nil {
		t.Fatalf("migrate SQLite models: %v", err)
	}
	oldDB, oldAppSetting := global.DBEngine, global.AppSetting
	global.DBEngine = db
	global.AppSetting = &setting.AppSettingS{DefaultPageSize: 10, MaxPageSize: 100}
	defer func() {
		global.DBEngine, global.AppSetting = oldDB, oldAppSetting
	}()
	if err := db.Create(&platform.Setting{Model: model.Model{}, Key: "allow_registration", Value: "true"}).Error; err != nil {
		t.Fatalf("enable member registration: %v", err)
	}
	article := model.BlogArticle{Model: &model.Model{}, Title: "欢迎留言", Desc: "desc", Content: "hello", State: 1}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article fixture: %v", err)
	}
	publicRouter := publicrouter.New()
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/members/register", strings.NewReader(`{"username":"member01","display_name":"读者","password":"correct-long-password"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	publicRouter.ServeHTTP(registerRecorder, registerReq)
	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf("member registration status = %d, body = %s", registerRecorder.Code, registerRecorder.Body.String())
	}
	if len(registerRecorder.Result().Cookies()) == 0 {
		t.Fatal("member registration did not set a session cookie")
	}
	memberCookie := registerRecorder.Result().Cookies()[0]

	createPayload := `{"content":"待审核评论"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/articles/"+strconv.FormatUint(uint64(article.ID), 10)+"/comments", strings.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(memberCookie)
	createRecorder := httptest.NewRecorder()
	publicRouter.ServeHTTP(createRecorder, createReq)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("comment create status = %d, body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created struct {
		ID uint32 `json:"id"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created comment response: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("comment create did not return an id")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/articles/"+strconv.FormatUint(uint64(article.ID), 10)+"/comments", nil)
	listRecorder := httptest.NewRecorder()
	publicRouter.ServeHTTP(listRecorder, listReq)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("public comments list status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	if bytes.Contains(listRecorder.Body.Bytes(), []byte("待审核评论")) {
		t.Fatal("pending comments should not appear in public list")
	}

	adminRouter := adminrouter.New()
	adminLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"username":"admin","password":"correct-horse-battery"}`))
	adminLoginReq.Header.Set("Content-Type", "application/json")
	adminLoginRecorder := httptest.NewRecorder()
	adminRouter.ServeHTTP(adminLoginRecorder, adminLoginReq)
	if adminLoginRecorder.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, body = %s", adminLoginRecorder.Code, adminLoginRecorder.Body.String())
	}
	adminCookies := adminLoginRecorder.Result().Cookies()
	if len(adminCookies) == 0 {
		t.Fatal("admin login did not set a cookie")
	}

	approveReq := httptest.NewRequest(http.MethodPatch, "/api/v1/comments/"+strconv.FormatUint(uint64(created.ID), 10)+"/state", bytes.NewBufferString(`{"state":1}`))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.AddCookie(adminCookies[0])
	approveRecorder := httptest.NewRecorder()
	adminRouter.ServeHTTP(approveRecorder, approveReq)
	if approveRecorder.Code != http.StatusOK {
		t.Fatalf("approve comment status = %d, body = %s", approveRecorder.Code, approveRecorder.Body.String())
	}

	publicListAfterReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/articles/"+strconv.FormatUint(uint64(article.ID), 10)+"/comments", nil)
	publicListAfterRecorder := httptest.NewRecorder()
	publicRouter.ServeHTTP(publicListAfterRecorder, publicListAfterReq)
	if publicListAfterRecorder.Code != http.StatusOK {
		t.Fatalf("public list after approval status = %d, body = %s", publicListAfterRecorder.Code, publicListAfterRecorder.Body.String())
	}
	if !bytes.Contains(publicListAfterRecorder.Body.Bytes(), []byte("待审核评论")) {
		t.Fatalf("approved comment not visible publicly: %s", publicListAfterRecorder.Body.String())
	}
}
