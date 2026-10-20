package main

import (
	"blog-server/pkg/app"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponse(t *testing.T) {
	router := gin.New()
	router.GET("/test/ping", func(c *gin.Context) {
		app.NewResponse(c).ToResponse(map[string]string{"msg": "ping pong is ok"})
	})

	request := httptest.NewRequest(http.MethodGet, "/test/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", recorder.Code)
	}
	if got, want := recorder.Body.String(), `{"msg":"ping pong is ok"}`; got != want {
		t.Fatalf("unexpected response body %q, want %q", got, want)
	}
}

func TestMultiHandler(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("message", "Hello, world!")
		c.Next()
	})
	router.GET("/hello", func(c *gin.Context) {
		c.Next()
	}, func(c *gin.Context) {
		message := c.MustGet("message").(string)
		c.String(http.StatusOK, message+" from world")
	})

	request := httptest.NewRequest(http.MethodGet, "/hello", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", recorder.Code)
	}
	if got, want := recorder.Body.String(), "Hello, world! from world"; got != want {
		t.Fatalf("unexpected response body %q, want %q", got, want)
	}
}
