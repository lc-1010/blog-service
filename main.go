package main

import (
	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/internal/routers"
	"blog-server/pkg/logger"
	"blog-server/pkg/setting"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"gopkg.in/natefinch/lumberjack.v2"
)

// @title 博客
// @version 1.0
// @description 博客后台
func main() {
	err := setupSetting()
	if err != nil {
		log.Fatalf("setupSetting err:%v", err)
	}
	err = setupDBEngine()
	if err != nil {
		log.Fatalf("setupDBEngine err:%v", err)
	}
	err = setupLogger()
	if err != nil {
		log.Fatalf("setupLogger err:%v", err)
	}

	gin.SetMode(global.ServerSetting.RunMode)

	router := routers.NewRouter()
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

var trans ut.Translator

func init() {
	uni := ut.New(zh.New())
	trans, _ = uni.GetTranslator("zh")
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		_ = zh_translations.RegisterDefaultTranslations(v, trans)
	}

}

// setupSetting
func setupSetting() error {
	setting, err := setting.NewSetting()
	if err != nil {
		return err
	}
	err = setting.ReadSection("Server", &global.ServerSetting)
	if err != nil {
		return err
	}
	err = setting.ReadSection("App", &global.AppSetting)
	if err != nil {
		return err
	}
	err = setting.ReadSection("Database", &global.DatabaseSetting)
	if err != nil {
		return err
	}
	global.ServerSetting.ReadTimeout *= time.Second
	global.ServerSetting.WriteTimeout *= time.Second
	return nil
}

func setupDBEngine() error {
	var err error
	// use = 设置启动，不能用:= 因为其他包调用时是会是nil
	global.DBEngine, err = model.NewDBEngine(global.DatabaseSetting)
	if err != nil {
		return err
	}
	return nil
}

func setupLogger() error {
	fileName := global.AppSetting.LogServePath + "/" +
		global.AppSetting.LogFileName + global.AppSetting.LogFileExt
	global.Logger = logger.NewLogger(&lumberjack.Logger{
		Filename:  fileName,
		MaxSize:   600,
		MaxAge:    10,
		LocalTime: true,
	}, "", log.LstdFlags).WithCaller(2)
	return nil
}
