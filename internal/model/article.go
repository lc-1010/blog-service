package model

import "blog-server/pkg/app"

type BlogArticle struct {
	*Model
	Title string `json:"title"`
	// 文章简述
	Desc string `json:"desc"`
	// 封面图片地址
	CoverImageUrl string `json:"cover_image_url"`
	Content       string `json:"content" gorm:"type:longtext"`
	TagID         uint32 `json:"tag_id" gorm:"-"`
	// 状态 0 禁用，1 启用
	State uint8 `json:"state"`
}

func (model BlogArticle) TableName() string {
	return "blog_article"
}

type CreateArticle struct {
	TagID         uint32 `json:"tag_id" binding:"required"`
	Title         string `json:"title" binding:"required,min=2,max=100"`
	Desc          string `json:"desc" binding:"max=255"`
	CoverImageUrl string `json:"cover_image_url" binding:"omitempty,url"`
	Content       string `json:"content" binding:"required,min=1"`
	CreatedBy     string `json:"created_by" binding:"required,min=2,max=100"`
	State         *uint8 `json:"state" binding:"omitempty,oneof=0 1"`
}

type UpdateArticle struct {
	TagID         uint32 `json:"tag_id" binding:"required"`
	Title         string `json:"title" binding:"required,min=2,max=100"`
	Desc          string `json:"desc" binding:"max=255"`
	CoverImageUrl string `json:"cover_image_url" binding:"omitempty,url"`
	Content       string `json:"content" binding:"required,min=1"`
	ModifiedBy    string `json:"modified_by" binding:"required,min=2,max=100"`
}

type UpdateArticleState struct {
	State      *uint8 `json:"state" binding:"required,oneof=0 1"`
	ModifiedBy string `json:"modified_by" binding:"required,min=2,max=100"`
}

type ArticleSwagger struct {
	List  []*BlogArticle
	Pager *app.Pager
}
