package test

import (
	"encoding/json"
	"fmt"
	"testing"
)

type BlogArticle struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	Desc          string `json:"desc"`
	CoverImageURL string `json:"cover_image_url"`
	Content       string `json:"content"`
	CreatedOn     uint   `json:"created_on"`
	CreatedBy     string `json:"created_by"`
	ModifiedOn    uint   `json:"modified_on"`
	ModifiedBy    string `json:"modified_by"`
	DeletedOn     uint   `json:"deleted_on"`
	IsDel         uint8  `json:"is_del"`
	State         uint8  `json:"state"`
}

func TestArticleCrate(t *testing.T) {
	article := BlogArticle{
		Title:         "文章标题",
		Desc:          "文章简述",
		CoverImageURL: "https://example.com/image.jpg",
		Content:       "文章内容",
		CreatedOn:     1629891178,
		CreatedBy:     "张三",
		State:         1,
	}

	data, err := json.Marshal(article)
	if err != nil {
		fmt.Println("JSON marshaling failed:", err)
		return
	}

	fmt.Printf("%s\n", data)
}
