package dao

import (
	"blog-server/internal/model"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTagTestDao(t *testing.T) *Dao {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:tag-dao-test?mode=memory&cache=shared"), &gorm.Config{})
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
	return New(db)
}

func TestTagCRUDWithSQLite(t *testing.T) {
	dao := newTagTestDao(t)
	if err := dao.CreateTag("go", 1, "tester"); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	tag, err := dao.CheckName("go")
	if err != nil {
		t.Fatalf("check tag name: %v", err)
	}
	if tag.ID == 0 || tag.CreatedBy != "tester" {
		t.Fatalf("unexpected created tag: %+v", tag)
	}

	count, err := dao.CountTag("go", 1)
	if err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	if err := dao.UpdateTag(tag.ID, "golang", 0, "editor"); err != nil {
		t.Fatalf("update tag: %v", err)
	}
	updated, err := dao.CheckName("golang")
	if err != nil {
		t.Fatalf("find updated tag: %v", err)
	}
	if updated.State != 0 || updated.ModifiedBy != "editor" {
		t.Fatalf("unexpected updated tag: %+v", updated)
	}

	if err := dao.DeleteTag(updated.ID); err != nil {
		t.Fatalf("delete tag: %v", err)
	}
	if _, err := dao.CheckName("golang"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("check deleted tag error = %v, want record not found", err)
	}
	var deleted model.BlogTag
	if err := dao.engine.Unscoped().First(&deleted, updated.ID).Error; err != nil {
		t.Fatalf("deleted tag should be retained: %v", err)
	}
	if deleted.IsDel != 1 || deleted.DeletedOn == 0 {
		t.Fatalf("tag was not soft-deleted: %+v", deleted)
	}
}

func TestUpdateTagReturnsDatabaseError(t *testing.T) {
	dao := newTagTestDao(t)
	sqlDB, err := dao.engine.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sqlite database: %v", err)
	}

	if err := dao.UpdateTag(1, "closed", 1, "tester"); err == nil {
		t.Fatal("expected update to report a closed database error")
	}
}
