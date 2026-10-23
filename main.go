package main

import (
	"blog-server/global"
	"blog-server/internal/bootstrap"
	publicrouter "blog-server/internal/routers/public"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @title 博客
// @version 1.0
// @description 博客公开服务
func main() {
	if err := bootstrap.Initialize(); err != nil {
		log.Fatalf("bootstrap.Initialize err:%v", err)
	}

	gin.SetMode(global.ServerSetting.RunMode)

	router := publicrouter.New()
	s := &http.Server{
		Addr:           ":" + global.ServerSetting.HttpPort,
		Handler:        router,
		ReadTimeout:    global.ServerSetting.ReadTimeout,  //10 * time.Second,
		WriteTimeout:   global.ServerSetting.WriteTimeout, //10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server.ListenAndServe err: %v", err)
	}
}
