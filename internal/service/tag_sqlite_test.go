package service

import (
	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/pkg/errcode"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTagTestService(t *testing.T) *Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:tag-service-test?mode=memory&cache=shared"), &gorm.Config{})
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
	previousDB := global.DBEngine
	global.DBEngine = db
	t.Cleanup(func() {
		global.DBEngine = previousDB
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite database: %v", err)
		}
	})
	svc := New(context.Background())
	return &svc
}

func TestCreateTagWithSQLite(t *testing.T) {
	svc := newTagTestService(t)
	param := &CrateTagRquest{Name: "go", State: 1, CreatedBy: "tester"}

	if err := svc.CrateTag(param); err != nil {
		t.Fatalf("create tag when name is absent: %v", err)
	}
	if err := svc.CrateTag(param); err != errcode.ErrorCrateTagExists {
		t.Fatalf("duplicate create error = %v, want %v", err, errcode.ErrorCrateTagExists)
	}
}
