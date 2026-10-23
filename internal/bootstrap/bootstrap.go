package bootstrap

import (
	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/pkg/logger"
	"blog-server/pkg/setting"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Initialize() error {
	return initialize("")
}

func InitializeAdmin() error {
	return initialize("-admin")
}

func initialize(logSuffix string) error {
	config, err := setting.NewSetting()
	if err != nil {
		return err
	}
	if err := config.ReadSection("Server", &global.ServerSetting); err != nil {
		return err
	}
	if err := config.ReadSection("App", &global.AppSetting); err != nil {
		return err
	}
	if err := config.ReadSection("Database", &global.DatabaseSetting); err != nil {
		return err
	}
	if value, ok := os.LookupEnv("BLOG_DB_HOST"); ok {
		global.DatabaseSetting.Host = value
	}
	if value, ok := os.LookupEnv("BLOG_DB_USERNAME"); ok {
		global.DatabaseSetting.UserName = value
	}
	if value, ok := os.LookupEnv("BLOG_DB_PASSWORD"); ok {
		global.DatabaseSetting.Password = value
	}
	if value, ok := os.LookupEnv("BLOG_DB_NAME"); ok {
		global.DatabaseSetting.DBName = value
	}

	global.ServerSetting.ReadTimeout *= time.Second
	global.ServerSetting.WriteTimeout *= time.Second

	global.DBEngine, err = model.NewDBEngine(global.DatabaseSetting)
	if err != nil {
		return err
	}

	fileName := filepath.Join(
		global.AppSetting.LogServePath,
		global.AppSetting.LogFileName+logSuffix+global.AppSetting.LogFileExt,
	)
	global.Logger = logger.NewLogger(&lumberjack.Logger{
		Filename:  fileName,
		MaxSize:   600,
		MaxAge:    10,
		LocalTime: true,
	}, "", log.LstdFlags).WithCaller(2)
	return nil
}
