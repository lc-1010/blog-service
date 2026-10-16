package routers

import (
	_ "blog-server/docs"
	"blog-server/internal/middleware"
	v1 "blog-server/internal/routers/api/v1"
	"blog-server/internal/routers/ping"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/locales/zh_Hant_TW"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// var trans ut.Translator
var v *validator.Validate
var uni *ut.UniversalTranslator

func init() {
	// 初始化语言设置
	uni = ut.New(en.New(), zh.New(), zh_Hant_TW.New())
	//trans, _ = uni.GetTranslator("zh")
	v, _ = binding.Validator.Engine().(*validator.Validate)
}

// NewRouter tags articles curd
func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	//传递翻译器
	r.Use(middleware.TranslateSet(uni, v))

	article := v1.NewArticle()
	tags := v1.NewTag()
	ping := ping.NewPing()

	p := r.Group("/test")
	{
		p.GET("/ping", ping.Pong)
	}
	apiv1 := r.Group("/api/v1")

	{
		//tags
		apiv1.POST("/tags", tags.Create)
		apiv1.DELETE("/tags/:id", tags.Delete)
		apiv1.PUT("/tags/:id", tags.Update)
		apiv1.PATCH("/tags/:id/state", tags.Update)
		apiv1.GET("/tags", tags.List)
		//articles
		apiv1.POST("/articles", article.Create)
		apiv1.GET("/articles", article.List)
		apiv1.GET("/articles/:id", article.Get)
		apiv1.PATCH("/articles/:id/state", article.Update)
		apiv1.PUT("/articles/:id", article.Update)
		apiv1.DELETE("/articles/:id", article.Delete)
	}
	// swag router
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
