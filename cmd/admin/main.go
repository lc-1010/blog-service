package main

import (
	"blog-server/global"
	"blog-server/internal/bootstrap"
	adminrouter "blog-server/internal/routers/admin"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := bootstrap.InitializeAdmin(); err != nil {
		log.Fatalf("bootstrap.Initialize err: %v", err)
	}
	if global.ServerSetting.AdminHttpPort == "" {
		log.Fatal("AdminHttpPort is not configured")
	}

	gin.SetMode(global.ServerSetting.RunMode)
	server := &http.Server{
		Addr:           ":" + global.ServerSetting.AdminHttpPort,
		Handler:        adminrouter.New(),
		ReadTimeout:    global.ServerSetting.ReadTimeout,
		WriteTimeout:   global.ServerSetting.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("admin server.ListenAndServe err: %v", err)
	}
}
