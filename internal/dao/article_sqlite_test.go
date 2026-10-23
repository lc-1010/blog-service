package dao

import (
	"blog-server/internal/model"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newArticleTestDao(t *testing.T) *Dao {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:article-dao-test?mode=memory&cache=shared"), &gorm.Config{})
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
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite database: %v", err)
		}
	})
	return New(db)
}

func createArticleTestTag(t *testing.T, dao *Dao, name string) uint32 {
	t.Helper()
	tag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: name, State: 1}
	if err := dao.engine.Create(&tag).Error; err != nil {
		t.Fatalf("create tag fixture: %v", err)
	}
	return tag.ID
}

func TestArticleCRUDWithSQLite(t *testing.T) {
	dao := newArticleTestDao(t)
	tagID := createArticleTestTag(t, dao, "sqlite")
	article := &model.BlogArticle{
		Model:   &model.Model{CreatedBy: "author"},
		Title:   "SQLite guide",
		Desc:    "A test article",
		Content: "Article body",
		State:   1,
	}
	if err := dao.CreateArticle(article, tagID); err != nil {
		t.Fatalf("create article: %v", err)
	}
	if article.ID == 0 {
		t.Fatal("created article has no ID")
	}

	got, err := dao.GetArticle(article.ID)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if got.Title != article.Title || got.Content != article.Content || got.TagID != tagID {
		t.Fatalf("unexpected article: %+v", got)
	}

	articles, total, err := dao.ListArticles("SQLite", tagID, 1, 0, 10)
	if err != nil {
		t.Fatalf("list articles: %v", err)
	}
	if total != 1 || len(articles) != 1 || articles[0].TagID != tagID {
		t.Fatalf("unexpected list result: total=%d articles=%+v", total, articles)
	}

	newTagID := createArticleTestTag(t, dao, "updated")
	if err := dao.UpdateArticle(article.ID, model.BlogArticle{
		Title:   "Updated title",
		Desc:    "Updated description",
		Content: "Updated body",
	}, newTagID, "editor"); err != nil {
		t.Fatalf("update article: %v", err)
	}
	got, err = dao.GetArticle(article.ID)
	if err != nil {
		t.Fatalf("get updated article: %v", err)
	}
	if got.Title != "Updated title" || got.TagID != newTagID || got.ModifiedBy != "editor" {
		t.Fatalf("unexpected updated article: %+v", got)
	}

	if err := dao.UpdateArticleState(article.ID, 0, "moderator"); err != nil {
		t.Fatalf("disable article: %v", err)
	}
	articles, total, err = dao.ListArticles("", 0, 0, 0, 10)
	if err != nil {
		t.Fatalf("list disabled articles: %v", err)
	}
	if total != 1 || len(articles) != 1 {
		t.Fatalf("disabled article missing: total=%d articles=%+v", total, articles)
	}
	if err := dao.UpdateArticleState(article.ID+100, 0, "moderator"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update missing article state error = %v, want record not found", err)
	}
	if err := dao.UpdateArticle(article.ID+100, model.BlogArticle{Title: "Missing"}, newTagID, "editor"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update missing article error = %v, want record not found", err)
	}

	if err := dao.DeleteArticle(article.ID, ""); err != nil {
		t.Fatalf("soft-delete article: %v", err)
	}
	if _, err := dao.GetArticle(article.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("get deleted article error = %v, want record not found", err)
	}
}

func TestCreateArticleRejectsInactiveOrMissingTag(t *testing.T) {
	dao := newArticleTestDao(t)
	article := &model.BlogArticle{Model: &model.Model{CreatedBy: "author"}, Title: "Article"}
	if err := dao.CreateArticle(article, 999); !errors.Is(err, ErrInvalidArticleTag) {
		t.Fatalf("create with missing tag error = %v, want invalid article tag", err)
	}

	tag := model.BlogTag{Model: &model.Model{CreatedBy: "tester"}, Name: "inactive", State: 0}
	if err := dao.engine.Create(&tag).Error; err != nil {
		t.Fatalf("create inactive tag fixture: %v", err)
	}
	if err := dao.CreateArticle(article, tag.ID); !errors.Is(err, ErrInvalidArticleTag) {
		t.Fatalf("create with inactive tag error = %v, want invalid article tag", err)
	}
}

func TestGetArticleReturnsDatabaseError(t *testing.T) {
	dao := newArticleTestDao(t)
	sqlDB, err := dao.engine.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sqlite database: %v", err)
	}
	if _, err := dao.GetArticle(1); err == nil {
		t.Fatal("expected get article to report a closed database error")
	}
}
