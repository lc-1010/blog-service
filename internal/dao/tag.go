package dao

import (
	"time"

	"blog-server/internal/model"
	"blog-server/pkg/app"
)

func (d *Dao) GetTag(id uint32, state uint8) (model.BlogTag, error) {
	tag := model.BlogTag{Model: &model.Model{ID: id}, State: state}
	return tag.Get(d.engine)
}

func (d *Dao) CountTag(name string, state uint8) (int, error) {
	tag := model.BlogTag{State: state, Name: name}
	return tag.Count(d.engine)
}

func (d *Dao) GetTagList(name string, state uint8, page, pageSize int) ([]*model.BlogTag, error) {
	tag := model.BlogTag{Name: name, State: state}
	pageOffset := app.GetPageOffset(page, pageSize)
	return tag.List(d.engine, pageOffset, pageSize)
}

func (d *Dao) GetTagListWithAnyState(name string, state *uint8, page, pageSize int) ([]*model.BlogTag, int, error) {
	query := d.engine.Model(&model.BlogTag{}).Where("is_del = ?", 0)
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if state != nil {
		query = query.Where("state = ?", *state)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tags []*model.BlogTag
	if err := query.Order("id DESC").Offset(app.GetPageOffset(page, pageSize)).Limit(pageSize).Find(&tags).Error; err != nil {
		return nil, 0, err
	}
	return tags, int(total), nil
}

func (d *Dao) CreateTag(name string, state uint8, crateBy string) error {
	tag := model.BlogTag{Name: name, State: state, Model: &model.Model{
		CreatedBy: crateBy,
	}}
	return tag.Create(d.engine)
}

func (d *Dao) CheckName(name string) (model.BlogTag, error) {

	tag := model.BlogTag{Name: name}
	return tag.CheckName(d.engine)
}

// UpdateTag  tag's function is db.Updates use values store key value
func (d *Dao) UpdateTag(id uint32, name string, state uint8, modifiedBy string) error {
	tag := model.BlogTag{Model: &model.Model{
		ID: id,
	}}
	values := map[string]any{
		"state":       state,
		"modified_by": modifiedBy,
	}
	if name != "" {
		values["name"] = name
	}

	return tag.Update(d.engine, values)
}
func (d *Dao) DeleteTag(id uint32) error {
	now := uint32(time.Now().Unix())
	return d.engine.Model(&model.BlogTag{}).
		Where("id = ? AND is_del = ?", id, 0).
		Updates(map[string]any{
			"is_del":      1,
			"deleted_on":  now,
			"modified_on": now,
		}).Error
}
