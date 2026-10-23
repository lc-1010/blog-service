// 文章管理
package v1

import (
	"blog-server/global"
	"blog-server/internal/model"
	"blog-server/internal/service"
	"blog-server/pkg/app"
	"blog-server/pkg/errcode"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Article struct{}

func NewArticle() Article {
	return Article{}
}

// @Summary 获取单篇文章
// @Produce json
// @Param id path int true "文章 ID"
// @Success 200 {object} model.BlogArticle "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 404 {object} errcode.Error "文章不存在"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles/{id} [get]
func (t Article) Get(c *gin.Context) {
	id, ok := articleID(c)
	if !ok {
		return
	}
	response := app.NewResponse(c)
	article, err := articleService(c).GetArticle(id)
	if err != nil {
		global.Logger.Errorf(c, "GetArticle(%d) err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorGetArticleFail)
		return
	}
	response.ToResponse(article)
}

func (t Article) PublicGet(c *gin.Context) {
	id, ok := articleID(c)
	if !ok {
		return
	}
	response := app.NewResponse(c)
	article, err := articleService(c).GetArticle(id)
	if err != nil {
		global.Logger.Errorf(c, "GetArticle(%d) err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorGetArticleFail)
		return
	}
	if article.State != 1 {
		response.ToErrorResponse(errcode.NotFound)
		return
	}
	response.ToResponse(article)
}

// @Summary 分页查询文章
// @Produce json
// @Param title query string false "标题关键字"
// @Param tag_id query int false "标签 ID"
// @Param state query int false "状态" Enums(0,1) default(1)
// @Param all_states query bool false "管理员查询全部状态"
// @Param page query int false "页码" minimum(1)
// @Param page_size query int false "每页数量" minimum(1) maximum(100)
// @Success 200 {object} model.ArticleSwagger "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles [get]
func (t Article) List(c *gin.Context) {
	param := service.ArticleListRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		global.Logger.Errorf(c, "bind article list params err: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	page := param.Page
	if page == 0 {
		page = app.GetPage(c)
	}
	pageSize := param.PageSize
	if pageSize == 0 {
		pageSize = app.GetPageSize(c)
	}
	articles, total, err := articleService(c).ListArticles(&param, page, pageSize)
	if err != nil {
		global.Logger.Errorf(c, "ListArticles err: %v", err)
		response.ToErrorResponse(errcode.ErrorGetArticleListFail)
		return
	}
	response.ToResponseList(articles, total)
}

func (t Article) PublicList(c *gin.Context) {
	param := service.ArticleListRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		global.Logger.Errorf(c, "bind public article list params err: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	page := param.Page
	if page == 0 {
		page = app.GetPage(c)
	}
	pageSize := param.PageSize
	if pageSize == 0 {
		pageSize = app.GetPageSize(c)
	}
	param.State = nil
	param.AllStates = false
	articles, total, err := articleService(c).ListArticles(&param, page, pageSize)
	if err != nil {
		global.Logger.Errorf(c, "List public articles err: %v", err)
		response.ToErrorResponse(errcode.ErrorGetArticleListFail)
		return
	}
	response.ToResponseList(articles, total)
}

// @Summary 创建文章
// @Accept json
// @Produce json
// @Param request body model.CreateArticle true "文章内容"
// @Success 200 {object} model.BlogArticle "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles [post]
func (t Article) Create(c *gin.Context) {
	param := model.CreateArticle{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		global.Logger.Errorf(c, "bind create article params err: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	article, err := articleService(c).CreateArticle(&param)
	if err != nil {
		global.Logger.Errorf(c, "CreateArticle err: %v", err)
		writeArticleError(response, err, errcode.ErrorCreateArticleFail)
		return
	}
	response.ToResponse(article)
}

// @Summary 全量更新文章
// @Accept json
// @Produce json
// @Param id path int true "文章 ID"
// @Param request body model.UpdateArticle true "文章内容"
// @Success 200 {object} model.BlogArticle "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 404 {object} errcode.Error "文章不存在"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles/{id} [put]
func (t Article) Update(c *gin.Context) {
	id, ok := articleID(c)
	if !ok {
		return
	}
	param := model.UpdateArticle{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		global.Logger.Errorf(c, "bind update article params err: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	if err := articleService(c).UpdateArticle(id, &param); err != nil {
		global.Logger.Errorf(c, "UpdateArticle(%d) err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorUpdateArticleFail)
		return
	}
	article, err := articleService(c).GetArticle(id)
	if err != nil {
		global.Logger.Errorf(c, "GetArticle(%d) after update err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorGetArticleFail)
		return
	}
	response.ToResponse(article)
}

// @Summary 更新文章状态
// @Accept json
// @Produce json
// @Param id path int true "文章 ID"
// @Param request body model.UpdateArticleState true "文章状态"
// @Success 200 {object} model.BlogArticle "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 404 {object} errcode.Error "文章不存在"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles/{id}/state [patch]
func (t Article) UpdateState(c *gin.Context) {
	id, ok := articleID(c)
	if !ok {
		return
	}
	param := model.UpdateArticleState{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		global.Logger.Errorf(c, "bind article state params err: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	if err := articleService(c).UpdateArticleState(id, &param); err != nil {
		global.Logger.Errorf(c, "UpdateArticleState(%d) err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorUpdateArticleFail)
		return
	}
	article, err := articleService(c).GetArticle(id)
	if err != nil {
		global.Logger.Errorf(c, "GetArticle(%d) after state update err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorGetArticleFail)
		return
	}
	response.ToResponse(article)
}

// @Summary 软删除文章
// @Produce json
// @Param id path int true "文章 ID"
// @Success 200 {object} map[string]string "成功"
// @Failure 400 {object} errcode.Error "请求错误"
// @Failure 404 {object} errcode.Error "文章不存在"
// @Failure 500 {object} errcode.Error "内部错误"
// @Router /api/v1/articles/{id} [delete]
func (t Article) Delete(c *gin.Context) {
	id, ok := articleID(c)
	if !ok {
		return
	}
	response := app.NewResponse(c)
	if err := articleService(c).DeleteArticle(id, ""); err != nil {
		global.Logger.Errorf(c, "DeleteArticle(%d) err: %v", id, err)
		writeArticleError(response, err, errcode.ErrorDeleteArticleFail)
		return
	}
	response.ToResponse(gin.H{"message": "deleted"})
}

func articleService(c *gin.Context) *service.Service {
	svc := service.New(c.Request.Context())
	return &svc
}

func articleID(c *gin.Context) (uint32, bool) {
	var uri struct {
		ID uint32 `uri:"id" binding:"required,gte=1"`
	}
	response := app.NewResponse(c)
	if err := c.ShouldBindUri(&uri); err != nil {
		global.Logger.Errorf(c, "bind article URI id err: %v", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(err.Error()))
		return 0, false
	}
	return uri.ID, true
}

func writeArticleError(response *app.Response, err error, fallback *errcode.Error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.ToErrorResponse(errcode.NotFound)
	case errors.Is(err, service.ErrInvalidArticleTag):
		response.ToErrorResponse(errcode.InvalidParams.WithDetails("tag_id must refer to an active tag"))
	default:
		response.ToErrorResponse(fallback)
	}
}
