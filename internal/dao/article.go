package dao

import (
	"blog-server/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidArticleTag = errors.New("article tag must exist and be active")

func (d *Dao) GetArticle(id uint32) (model.BlogArticle, error) {
	var article model.BlogArticle
	err := d.engine.Where("id = ? AND is_del = ?", id, 0).First(&article).Error
	if err != nil {
		return article, err
	}
	if err := d.engine.Model(&model.BlogArticleTag{}).
		Select("tag_id").
		Where("article_id = ? AND is_del = ?", id, 0).
		Limit(1).
		Scan(&article.TagID).Error; err != nil {
		return article, err
	}
	return article, nil
}

func (d *Dao) ListArticles(title string, tagID uint32, state uint8, pageOffset, pageSize int) ([]*model.BlogArticle, int, error) {
	return d.listArticles(title, tagID, &state, pageOffset, pageSize)
}

func (d *Dao) ListArticlesWithAnyState(title string, tagID uint32, state *uint8, pageOffset, pageSize int) ([]*model.BlogArticle, int, error) {
	return d.listArticles(title, tagID, state, pageOffset, pageSize)
}

func (d *Dao) listArticles(title string, tagID uint32, state *uint8, pageOffset, pageSize int) ([]*model.BlogArticle, int, error) {
	query := d.articleQueryWithState(title, tagID, state)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var articles []*model.BlogArticle
	if err := d.articleQueryWithState(title, tagID, state).
		Order("id DESC").
		Offset(pageOffset).
		Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, 0, err
	}
	if len(articles) == 0 {
		return articles, int(count), nil
	}

	articleIDs := make([]uint32, 0, len(articles))
	for _, article := range articles {
		articleIDs = append(articleIDs, article.ID)
	}
	var links []model.BlogArticleTag
	if err := d.engine.Where("article_id IN ? AND is_del = ?", articleIDs, 0).Find(&links).Error; err != nil {
		return nil, 0, err
	}
	tagIDs := make(map[uint32]uint32, len(links))
	for _, link := range links {
		tagIDs[link.ArticleID] = link.TagID
	}
	for _, article := range articles {
		article.TagID = tagIDs[article.ID]
	}
	return articles, int(count), nil
}

func (d *Dao) CreateArticle(article *model.BlogArticle, tagID uint32) error {
	return d.engine.Transaction(func(tx *gorm.DB) error {
		if err := activeTagExists(tx, tagID); err != nil {
			return err
		}
		if err := tx.Create(article).Error; err != nil {
			return err
		}
		link := model.BlogArticleTag{
			Model:     &model.Model{CreatedBy: article.CreatedBy},
			TagID:     tagID,
			ArticleID: article.ID,
		}
		return tx.Create(&link).Error
	})
}

func (d *Dao) UpdateArticle(id uint32, article model.BlogArticle, tagID uint32, modifiedBy string) error {
	return d.engine.Transaction(func(tx *gorm.DB) error {
		var existing model.BlogArticle
		if err := tx.Select("id").Where("id = ? AND is_del = ?", id, 0).First(&existing).Error; err != nil {
			return err
		}
		if err := activeTagExists(tx, tagID); err != nil {
			return err
		}
		result := tx.Model(&model.BlogArticle{Model: &model.Model{}}).
			Where("id = ? AND is_del = ?", id, 0).
			Updates(map[string]any{
				"title":           article.Title,
				"desc":            article.Desc,
				"cover_image_url": article.CoverImageUrl,
				"content":         article.Content,
				"modified_by":     modifiedBy,
			})
		if result.Error != nil {
			return result.Error
		}

		if err := tx.Where("article_id = ?", id).Delete(&model.BlogArticleTag{Model: &model.Model{}}).Error; err != nil {
			return err
		}
		link := model.BlogArticleTag{
			Model:     &model.Model{CreatedBy: modifiedBy},
			TagID:     tagID,
			ArticleID: id,
		}
		return tx.Create(&link).Error
	})
}

func (d *Dao) UpdateArticleState(id uint32, state uint8, modifiedBy string) error {
	return d.engine.Transaction(func(tx *gorm.DB) error {
		var existing model.BlogArticle
		if err := tx.Select("id").Where("id = ? AND is_del = ?", id, 0).First(&existing).Error; err != nil {
			return err
		}
		return tx.Model(&model.BlogArticle{Model: &model.Model{}}).
			Where("id = ? AND is_del = ?", id, 0).
			Updates(map[string]any{"state": state, "modified_by": modifiedBy}).Error
	})
}

func (d *Dao) DeleteArticle(id uint32, modifiedBy string) error {
	updates := map[string]any{
		"is_del":     1,
		"deleted_on": uint32(time.Now().Unix()),
	}
	if modifiedBy != "" {
		updates["modified_by"] = modifiedBy
	}
	return d.engine.Transaction(func(tx *gorm.DB) error {
		var existing model.BlogArticle
		if err := tx.Select("id").Where("id = ? AND is_del = ?", id, 0).First(&existing).Error; err != nil {
			return err
		}
		return tx.Model(&model.BlogArticle{Model: &model.Model{}}).
			Where("id = ? AND is_del = ?", id, 0).
			Updates(updates).Error
	})
}

func (d *Dao) articleQueryWithState(title string, tagID uint32, state *uint8) *gorm.DB {
	query := d.engine.Model(&model.BlogArticle{}).Where("blog_article.is_del = ?", 0)
	if state != nil {
		query = query.Where("blog_article.state = ?", *state)
	}
	if title != "" {
		query = query.Where("blog_article.title LIKE ?", "%"+title+"%")
	}
	if tagID != 0 {
		query = query.Where(
			"blog_article.id IN (?)",
			d.engine.Model(&model.BlogArticleTag{}).
				Select("article_id").
				Where("tag_id = ? AND is_del = ?", tagID, 0),
		)
	}
	return query
}

type DashboardStats struct {
	Articles  int64 `json:"articles"`
	Published int64 `json:"published"`
	Drafts    int64 `json:"drafts"`
	Tags      int64 `json:"tags"`
}

func (d *Dao) GetDashboardStats() (DashboardStats, error) {
	var stats DashboardStats
	if err := d.engine.Model(&model.BlogArticle{}).Where("is_del = ?", 0).Count(&stats.Articles).Error; err != nil {
		return DashboardStats{}, err
	}
	if err := d.engine.Model(&model.BlogArticle{}).Where("is_del = ? AND state = ?", 0, 1).Count(&stats.Published).Error; err != nil {
		return DashboardStats{}, err
	}
	if err := d.engine.Model(&model.BlogArticle{}).Where("is_del = ? AND state = ?", 0, 0).Count(&stats.Drafts).Error; err != nil {
		return DashboardStats{}, err
	}
	if err := d.engine.Model(&model.BlogTag{}).Where("is_del = ?", 0).Count(&stats.Tags).Error; err != nil {
		return DashboardStats{}, err
	}
	return stats, nil
}

func activeTagExists(tx *gorm.DB, tagID uint32) error {
	var tag model.BlogTag
	err := tx.Where("id = ? AND state = ? AND is_del = ?", tagID, 1, 0).First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrInvalidArticleTag
	}
	return err
}
