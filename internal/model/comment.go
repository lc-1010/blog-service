package model

// BlogComment stores member discussions attached to an article.
type BlogComment struct {
	*Model
	ArticleID uint32 `json:"article_id" gorm:"index;not null"`
	MemberID  uint32 `json:"member_id" gorm:"index;not null"`
	ParentID  uint32 `json:"parent_id" gorm:"default:0;index"`
	Content   string `json:"content" gorm:"type:text;not null"`
	State     uint8  `json:"state" gorm:"default:0;index"`
}

func (BlogComment) TableName() string {
	return "blog_article_comments"
}

type CreateComment struct {
	ParentID uint32 `json:"parent_id" binding:"omitempty,gte=0"`
	Content  string `json:"content" binding:"required,min=1,max=2000"`
}

type UpdateCommentState struct {
	State *uint8 `json:"state" binding:"required,oneof=0 1 2"`
}
