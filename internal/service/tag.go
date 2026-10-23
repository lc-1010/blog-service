package service

import (
	"errors"

	"blog-server/internal/model"
	"blog-server/pkg/app"
	"blog-server/pkg/errcode"

	"gorm.io/gorm"
)

type ConutTagRequest struct {
	Name  string `form:"name,omitempty" binding:"max=100"`
	State uint8  `form:"state,default=1" binding:"oneof=0 1" `
}

type GetTagListRequest struct {
	Name      string `form:"name,omitempty" binding:"max=100" json:"name"`
	State     uint8  `form:"state,default=1" binding:"oneof=0 1" json:"state"`
	AllStates bool   `form:"all_states" json:"all_states"`
}

type CrateTagRquest struct {
	Name      string `form:"name" binding:"required,min=1,max=100" json:"name"`
	CreatedBy string `form:"created_by" binding:"required,min=3,max=100" json:"created_by"`
	State     uint8  `form:"state,default=1" binding:"oneof=0 1" json:"state"`
}

type UpdateTagRequest struct {
	ID         uint32 `json:"-" form:"-" binding:"-"`
	Name       string `form:"name" binding:"required,min=1,max=100" json:"name"`
	State      uint8  `form:"state,default=1" binding:"oneof=0 1" json:"state"`
	ModifiedBy string `form:"modified_by" binding:"required,min=3,max=100" json:"modified_by"`
}

type DeleteTagRequest struct {
	ID uint32 `uri:"id" binding:"required,gte=1"`
}

/********Service ********/

func (svc *Service) CountTag(param *ConutTagRequest) (int, error) {
	return svc.dao.CountTag(param.Name, param.State)
}
func (svc *Service) GetTagList(param *GetTagListRequest, pager *app.Pager) ([]*model.BlogTag, error) {
	return svc.dao.GetTagList(param.Name, param.State, pager.Page, pager.PageSize)
}

func (svc *Service) GetTagListWithAnyState(param *GetTagListRequest, pager *app.Pager) ([]*model.BlogTag, int, error) {
	state := &param.State
	if param.AllStates {
		state = nil
	}
	return svc.dao.GetTagListWithAnyState(param.Name, state, pager.Page, pager.PageSize)
}

func (svc *Service) CrateTag(param *CrateTagRquest) error {
	tag, err := svc.dao.CheckName(param.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && tag.ID != 0 {
		return errcode.ErrorCrateTagExists
	}
	return svc.dao.CreateTag(param.Name, param.State, param.CreatedBy)
}
func (svc *Service) DeleteTag(param *DeleteTagRequest) error {
	return svc.dao.DeleteTag(param.ID)
}
func (svc *Service) UpdateTag(param *UpdateTagRequest) error {
	return svc.dao.UpdateTag(param.ID, param.Name, param.State, param.ModifiedBy)
}
