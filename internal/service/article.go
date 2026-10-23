package service

import (
	"blog-server/internal/dao"
	"blog-server/internal/model"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type ArticleListRequest struct {
	Title     string `form:"title" binding:"max=100"`
	TagID     uint32 `form:"tag_id" binding:"omitempty,gte=1"`
	State     *uint8 `form:"state" binding:"omitempty,oneof=0 1"`
	AllStates bool   `form:"all_states"`
	Page      int    `form:"page" binding:"omitempty,gte=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
}

func (svc *Service) GetArticle(id uint32) (model.BlogArticle, error) {
	return svc.dao.GetArticle(id)
}

func (svc *Service) ListArticles(param *ArticleListRequest, page, pageSize int) ([]*model.BlogArticle, int, error) {
	state := param.State
	if !param.AllStates && state == nil {
		defaultState := uint8(1)
		state = &defaultState
	}
	return svc.dao.ListArticlesWithAnyState(strings.TrimSpace(param.Title), param.TagID, state, (page-1)*pageSize, pageSize)
}

func (svc *Service) CreateArticle(param *model.CreateArticle) (*model.BlogArticle, error) {
	if _, err := svc.dao.GetTag(param.TagID, 1); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidArticleTag
		}
		return nil, err
	}
	state := uint8(1)
	if param.State != nil {
		state = *param.State
	}
	article := &model.BlogArticle{
		Model:         &model.Model{CreatedBy: param.CreatedBy},
		Title:         strings.TrimSpace(param.Title),
		Desc:          strings.TrimSpace(param.Desc),
		CoverImageUrl: strings.TrimSpace(param.CoverImageUrl),
		Content:       param.Content,
		State:         state,
		TagID:         param.TagID,
	}
	if err := svc.dao.CreateArticle(article, param.TagID); err != nil {
		return nil, normalizeArticleError(err)
	}
	return article, nil
}

func (svc *Service) UpdateArticle(id uint32, param *model.UpdateArticle) error {
	return normalizeArticleError(svc.dao.UpdateArticle(id, model.BlogArticle{
		Title:         strings.TrimSpace(param.Title),
		Desc:          strings.TrimSpace(param.Desc),
		CoverImageUrl: strings.TrimSpace(param.CoverImageUrl),
		Content:       param.Content,
	}, param.TagID, param.ModifiedBy))
}

func (svc *Service) UpdateArticleState(id uint32, param *model.UpdateArticleState) error {
	return svc.dao.UpdateArticleState(id, *param.State, param.ModifiedBy)
}

func (svc *Service) DeleteArticle(id uint32, modifiedBy string) error {
	return svc.dao.DeleteArticle(id, modifiedBy)
}

func (svc *Service) GetDashboardStats() (dao.DashboardStats, error) {
	return svc.dao.GetDashboardStats()
}

var ErrInvalidArticleTag = errors.New("article tag must exist and be active")

func normalizeArticleError(err error) error {
	if errors.Is(err, dao.ErrInvalidArticleTag) {
		return ErrInvalidArticleTag
	}
	return err
}
