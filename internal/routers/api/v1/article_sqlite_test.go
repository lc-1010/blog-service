package v1_test

import (
	"blog-server/global"
	"blog-server/internal/model"
	adminrouter "blog-server/internal/routers/admin"
	publicrouter "blog-server/internal/routers/public"
	"blog-server/pkg/logger"
	"blog-server/pkg/setting"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newArticleAPITest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:article-api-%p?mode=memory&cache=shared", t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.BlogTag{}, &model.BlogArticle{}, &model.BlogArticleTag{}); err != nil {
		t.Fatalf("migrate article tables: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	previousDB, previousLogger, previousAppSetting := global.DBEngine, global.Logger, global.AppSetting
	global.DBEngine = db
	global.Logger = logger.NewLogger(httptest.NewRecorder(), "", 0)
	global.AppSetting = &setting.AppSettingS{DefaultPageSize: 10, MaxPageSize: 100}
	t.Setenv("BLOG_ADMIN_USERNAME", "api-test-admin")
	t.Setenv("BLOG_ADMIN_PASSWORD", "api-test-password")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "api-test-session-secret-at-least-32-bytes")
	t.Cleanup(func() {
		global.DBEngine, global.Logger, global.AppSetting = previousDB, previousLogger, previousAppSetting
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite database: %v", err)
		}
	})
	return db, adminrouter.New()
}

func TestArticleAPICRUDWithSQLite(t *testing.T) {
	db, router := newArticleAPITest(t)
	tag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: "sqlite", State: 1}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag fixture: %v", err)
	}

	createBody := fmt.Sprintf(`{"tag_id":%d,"title":"SQLite guide","desc":"Summary","content":"Article body","created_by":"author"}`, tag.ID)
	recorder := performArticleRequest(router, http.MethodPost, "/api/v1/articles", createBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var created model.BlogArticle
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 || created.State != 1 || created.TagID != tag.ID {
		t.Fatalf("unexpected create response: %+v", created)
	}
	id := created.ID

	recorder = performArticleRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", id), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = performArticleRequest(router, http.MethodGet, "/api/v1/articles?title=SQLite&tag_id=1", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	updateBody := fmt.Sprintf(`{"tag_id":%d,"title":"Updated guide","desc":"Updated","content":"Updated body","modified_by":"editor"}`, tag.ID)
	recorder = performArticleRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/articles/%d", id), updateBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = performArticleRequest(router, http.MethodPatch, fmt.Sprintf("/api/v1/articles/%d/state", id), `{"state":0,"modified_by":"editor"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("state update status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	recorder = performArticleRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", id), "")
	var updated model.BlogArticle
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	if updated.State != 0 || updated.Title != "Updated guide" {
		t.Fatalf("unexpected updated article: %+v", updated)
	}
	recorder = performArticleRequest(router, http.MethodGet, "/api/v1/articles?all_states=true&page=1&page_size=10", "")
	var adminArticleResponse struct {
		List []*model.BlogArticle `json:"list"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &adminArticleResponse); err != nil {
		t.Fatalf("decode all-state article list: %v", err)
	}
	if recorder.Code != http.StatusOK || len(adminArticleResponse.List) != 1 || adminArticleResponse.List[0].State != 0 {
		t.Fatalf("admin article list did not include the draft: status=%d list=%+v", recorder.Code, adminArticleResponse.List)
	}

	recorder = performArticleRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", id+100), "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing article status = %d, want %d; body = %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	recorder = performArticleRequest(router, http.MethodGet, "/api/v1/articles/not-an-id", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid ID status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	recorder = performArticleRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/articles/%d", id), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	recorder = performArticleRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", id), "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("get deleted article status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestCreateArticleRejectsInvalidTagAndBody(t *testing.T) {
	_, router := newArticleAPITest(t)
	recorder := performArticleRequest(router, http.MethodPost, "/api/v1/articles", `{"tag_id":999,"title":"Article","content":"Body","created_by":"author"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid tag status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	recorder = performArticleRequest(router, http.MethodPost, "/api/v1/articles", `{"tag_id":1,"title":"","content":"Body","created_by":"author"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestPublicAPIOnlyReturnsPublishedContent(t *testing.T) {
	db, router := newArticleAPITest(t)
	publicRouter := publicrouter.New()
	activeTag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: "published", State: 1}
	inactiveTag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: "inactive", State: 0}
	if err := db.Create(&activeTag).Error; err != nil {
		t.Fatalf("create active tag: %v", err)
	}
	if err := db.Create(&inactiveTag).Error; err != nil {
		t.Fatalf("create inactive tag: %v", err)
	}

	published := model.BlogArticle{
		Model: &model.Model{CreatedBy: "author"},
		Title: "Public article",
		State: 1,
	}
	draft := model.BlogArticle{
		Model: &model.Model{CreatedBy: "author"},
		Title: "Private draft",
		State: 0,
	}
	if err := db.Create(&published).Error; err != nil {
		t.Fatalf("create published article: %v", err)
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("create draft article: %v", err)
	}
	for _, item := range []struct {
		articleID uint32
		tagID     uint32
	}{
		{articleID: published.ID, tagID: activeTag.ID},
		{articleID: draft.ID, tagID: inactiveTag.ID},
	} {
		link := model.BlogArticleTag{
			Model:     &model.Model{CreatedBy: "author"},
			ArticleID: item.articleID,
			TagID:     item.tagID,
		}
		if err := db.Create(&link).Error; err != nil {
			t.Fatalf("create article tag: %v", err)
		}
	}

	recorder := performArticleRequest(router, http.MethodGet, "/api/v1/tags?all_states=true&page=1&page_size=10", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin tag list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var adminTagResponse struct {
		List []*model.BlogTag `json:"list"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &adminTagResponse); err != nil {
		t.Fatalf("decode all-state tag list: %v", err)
	}
	if len(adminTagResponse.List) != 2 {
		t.Fatalf("admin tag list returned %d tags, want 2", len(adminTagResponse.List))
	}

	recorder = performArticleRequest(publicRouter, http.MethodGet, "/api/v1/public/articles?all_states=true&state=0", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("public list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var articleResponse struct {
		List []*model.BlogArticle `json:"list"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &articleResponse); err != nil {
		t.Fatalf("decode public article list: %v", err)
	}
	if len(articleResponse.List) != 1 || articleResponse.List[0].ID != published.ID {
		t.Fatalf("public API returned unexpected articles: %+v", articleResponse.List)
	}

	recorder = performArticleRequest(publicRouter, http.MethodGet, fmt.Sprintf("/api/v1/public/articles/%d", draft.ID), "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("public draft detail status = %d, want %d; body = %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}

	recorder = performArticleRequest(publicRouter, http.MethodGet, "/api/v1/public/tags?state=0&all_states=true", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("public tags status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var tagResponse struct {
		List []*model.BlogTag `json:"list"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &tagResponse); err != nil {
		t.Fatalf("decode public tags: %v", err)
	}
	if len(tagResponse.List) != 1 || tagResponse.List[0].ID != activeTag.ID {
		t.Fatalf("public API returned unexpected tags: %+v", tagResponse.List)
	}
}

func TestEmbeddedFrontendPagesAreServed(t *testing.T) {
	publicRouter := publicrouter.New()
	for _, path := range []string{"/", "/assets/public.css", "/assets/public.js"} {
		recorder := httptest.NewRecorder()
		publicRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.Len() == 0 {
			t.Errorf("GET %s returned an empty response", path)
		}
	}
	adminRouter := adminrouter.New()
	for _, path := range []string{"/admin/", "/admin/style.css", "/admin/app.js"} {
		recorder := httptest.NewRecorder()
		adminRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.Len() == 0 {
			t.Errorf("GET %s returned an empty response", path)
		}
	}
}

func TestArticleAPIReportsDatabaseFailure(t *testing.T) {
	db, router := newArticleAPITest(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sqlite database: %v", err)
	}
	recorder := performArticleRequest(router, http.MethodGet, "/api/v1/articles/1", "")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("database error status = %d, want %d; body = %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
}

func performArticleRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.HasPrefix(path, "/api/v1/") && !strings.HasPrefix(path, "/api/v1/public/") {
		request.AddCookie(adminSessionCookie(router))
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func adminSessionCookie(router http.Handler) *http.Cookie {
	login := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/login",
		bytes.NewBufferString(`{"username":"api-test-admin","password":"api-test-password"}`),
	)
	login.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, login)
	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "blog_admin_session" {
			return cookie
		}
	}
	return nil
}
