package v1_test

import (
	"blog-server/global"
	"blog-server/internal/model"
	adminrouter "blog-server/internal/routers/admin"
	"blog-server/pkg/logger"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateTagUsesURIIDWithSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tag-api-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.BlogTag{}); err != nil {
		t.Fatalf("migrate blog tags: %v", err)
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

	previousDB, previousLogger := global.DBEngine, global.Logger
	global.DBEngine = db
	global.Logger = logger.NewLogger(httptest.NewRecorder(), "", 0)
	t.Setenv("BLOG_ADMIN_USERNAME", "api-test-admin")
	t.Setenv("BLOG_ADMIN_PASSWORD", "api-test-password")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "api-test-session-secret-at-least-32-bytes")
	t.Cleanup(func() {
		global.DBEngine, global.Logger = previousDB, previousLogger
	})

	tag := model.BlogTag{
		Model: &model.Model{CreatedBy: "tester"},
		Name:  "before",
		State: 1,
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag fixture: %v", err)
	}

	router := adminrouter.New()
	sessionCookie := adminSessionCookie(router)
	values := url.Values{
		"name":        {"after"},
		"state":       {"0"},
		"modified_by": {"tester"},
	}
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/tags/"+strconv.FormatUint(uint64(tag.ID), 10),
		strings.NewReader(values.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(sessionCookie)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var updated model.BlogTag
	if err := db.First(&updated, tag.ID).Error; err != nil {
		t.Fatalf("load updated tag: %v", err)
	}
	if updated.Name != "after" || updated.State != 0 {
		t.Fatalf("unexpected tag after update: %+v", updated)
	}

	values.Set("state", "2")
	request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/tags/"+strconv.FormatUint(uint64(tag.ID), 10),
		strings.NewReader(values.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(sessionCookie)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d, want %d; body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}
