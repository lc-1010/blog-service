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
	Name  string `form:"name,omitempty" binding:"max=100"`
	State uint8  `form:"state,default=1" binding:"oneof=0 1" `
}

type CrateTagRquest struct {
	Name      string `form:"name,omitempty" binding:"max=100"`
	CreatedBy string `form:"created_by" binding:"required,min=3,max=100"`
	State     uint8  `form:"state,default=1" binding:"oneof=0 1" `
}

type UpdateTagRequest struct {
	ID         uint32 `json:"-" form:"-" binding:"-"`
	Name       string `form:"name,omitempty" binding:"max=100"`
	State      uint8  `form:"state,default=1" binding:"oneof=0 1" `
	ModifiedBy string `form:"modified_by" binding:"required,min=3,max=100"`
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
