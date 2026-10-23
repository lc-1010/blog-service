package admin

import (
	"blog-server/internal/routers/public"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminServiceServesUIAndProtectsManagementAPI(t *testing.T) {
	t.Setenv("BLOG_ADMIN_USERNAME", "router-test-admin")
	t.Setenv("BLOG_ADMIN_PASSWORD", "router-test-password")
	t.Setenv("BLOG_ADMIN_SESSION_SECRET", "router-test-session-secret-at-least-32-bytes")

	router := New()
	for _, test := range []struct {
		path       string
		statusCode int
		wantBody   string
	}{
		{path: "/admin/", statusCode: http.StatusOK, wantBody: "纸间 · 平台管理后台"},
		{path: "/admin/app.js", statusCode: http.StatusOK, wantBody: "/api/v1/admin/platform"},
		{path: "/admin/style.css", statusCode: http.StatusOK, wantBody: ".sidebar"},
		{path: "/api/v1/articles", statusCode: http.StatusUnauthorized, wantBody: "authentication required"},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != test.statusCode {
			t.Errorf("GET %s status = %d, want %d; body = %s", test.path, recorder.Code, test.statusCode, recorder.Body.String())
			continue
		}
		if !strings.Contains(recorder.Body.String(), test.wantBody) {
			t.Errorf("GET %s body does not contain %q", test.path, test.wantBody)
		}
	}
}

func TestPublicServiceDoesNotExposeAdminRoutes(t *testing.T) {
	router := public.New()
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/admin/"},
		{method: http.MethodPost, path: "/api/v1/admin/login"},
		{method: http.MethodGet, path: "/api/v1/articles"},
		{method: http.MethodPost, path: "/api/v1/articles"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, http.StatusNotFound)
		}
	}
}
