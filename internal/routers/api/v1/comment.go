package v1

import (
	"blog-server/global"
	"blog-server/internal/membership"
	"blog-server/internal/model"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Comment struct{}

func NewComment() Comment { return Comment{} }

func (t Comment) PublicList(c *gin.Context) {
	articleID, ok := publicCommentArticleID(c)
	if !ok {
		return
	}
	db := global.DBEngine
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return
	}
	if !publishedArticleExists(c, db, articleID) {
		return
	}
	parentID := uint32(0)
	if value := strings.TrimSpace(c.Query("parent_id")); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid parent_id"})
			return
		}
		parentID = uint32(parsed)
	}
	var comments []model.BlogComment
	if err := db.Where("article_id = ? AND is_del = ? AND state = ? AND parent_id = ?", articleID, 0, 1, parentID).
		Order("created_on ASC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load comments"})
		return
	}
	result := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		member := membership.Member{}
		if err := db.Select("username", "display_name").Where("id = ?", comment.MemberID).First(&member).Error; err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load comment author"})
			return
		}
		result = append(result, gin.H{
			"id":           comment.ID,
			"article_id":   comment.ArticleID,
			"parent_id":    comment.ParentID,
			"member_id":    comment.MemberID,
			"username":     member.Username,
			"display_name": member.DisplayName,
			"content":      comment.Content,
			"state":        comment.State,
			"created_on":   comment.CreatedOn,
		})
	}
	c.JSON(http.StatusOK, gin.H{"list": result})
}

func (t Comment) Create(c *gin.Context) {
	articleID, ok := publicCommentArticleID(c)
	if !ok {
		return
	}
	memberID, ok := currentMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
		return
	}
	var input model.CreateComment
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "content is required"})
		return
	}
	db := global.DBEngine
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return
	}
	var article model.BlogArticle
	if err := db.Where("id = ? AND is_del = ? AND state = ?", articleID, 0, 1).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load article"})
		}
		return
	}
	if input.ParentID != 0 {
		var parent model.BlogComment
		if err := db.Where("id = ? AND article_id = ? AND is_del = ?", input.ParentID, articleID, 0).First(&parent).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"message": "parent comment not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load parent comment"})
			}
			return
		}
	}
	comment := model.BlogComment{
		Model:     &model.Model{},
		ArticleID: articleID,
		MemberID:  memberID,
		ParentID:  input.ParentID,
		Content:   strings.TrimSpace(input.Content),
		State:     0,
	}
	if err := db.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create comment"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         comment.ID,
		"article_id": comment.ArticleID,
		"parent_id":  comment.ParentID,
		"state":      comment.State,
		"created_on": comment.CreatedOn,
		"content":    comment.Content,
	})
}

func (t Comment) List(c *gin.Context) {
	db := global.DBEngine
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return
	}
	var articleID uint32
	if value := strings.TrimSpace(c.Query("article_id")); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid article_id"})
			return
		}
		articleID = uint32(parsed)
	}
	query := db.Model(&model.BlogComment{}).Where("is_del = ?", 0)
	if articleID != 0 {
		query = query.Where("article_id = ?", articleID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to count comments"})
		return
	}
	page, pageSize := boundedPage(c.Query("page"), c.Query("page_size"))
	var comments []model.BlogComment
	if err := query.Order("created_on DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list comments"})
		return
	}
	result := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		member := membership.Member{}
		if err := db.Select("username", "display_name").Where("id = ?", comment.MemberID).First(&member).Error; err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load comment author"})
			return
		}
		result = append(result, gin.H{
			"id":           comment.ID,
			"article_id":   comment.ArticleID,
			"parent_id":    comment.ParentID,
			"member_id":    comment.MemberID,
			"username":     member.Username,
			"display_name": member.DisplayName,
			"content":      comment.Content,
			"state":        comment.State,
			"created_on":   comment.CreatedOn,
		})
	}
	c.JSON(http.StatusOK, gin.H{"list": result, "pager": gin.H{"page": page, "page_size": pageSize, "total_rows": total}})
}

func (t Comment) UpdateState(c *gin.Context) {
	var uri struct {
		ID uint32 `uri:"id" binding:"required,gte=1"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid comment id"})
		return
	}
	var input model.UpdateCommentState
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if err := c.ShouldBindJSON(&input); err != nil || input.State == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "state is required"})
		return
	}
	db := global.DBEngine
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "database is not available"})
		return
	}
	result := db.Model(&model.BlogComment{}).Where("id = ? AND is_del = ?", uri.ID, 0).Update("state", *input.State)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update comment state"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "comment not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment state updated"})
}

func publishedArticleExists(c *gin.Context, db *gorm.DB, articleID uint32) bool {
	var article model.BlogArticle
	err := db.Select("id").Where("id = ? AND is_del = ? AND state = ?", articleID, 0, 1).First(&article).Error
	if err == nil {
		return true
	}
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		return false
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load article"})
	return false
}

func publicCommentArticleID(c *gin.Context) (uint32, bool) {
	parsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || parsed == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("invalid article id: %v", err)})
		return 0, false
	}
	return uint32(parsed), true
}

func currentMemberID(c *gin.Context) (uint32, bool) {
	value, ok := c.Get("member_id")
	if !ok {
		return 0, false
	}
	memberID, ok := value.(uint32)
	return memberID, ok
}

func boundedPage(pageValue, pageSizeValue string) (int, int) {
	page := 1
	if parsed, err := strconv.Atoi(pageValue); err == nil && parsed > 0 {
		page = parsed
	}
	pageSize := 20
	if parsed, err := strconv.Atoi(pageSizeValue); err == nil && parsed > 0 {
		pageSize = parsed
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
