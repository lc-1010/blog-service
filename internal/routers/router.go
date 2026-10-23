package routers

import (
	"blog-server/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/locales/zh_Hant_TW"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
)

var v *validator.Validate
var uni *ut.UniversalTranslator

func init() {
	uni = ut.New(en.New(), zh.New(), zh_Hant_TW.New())
	v, _ = binding.Validator.Engine().(*validator.Validate)
}

func NewEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.TranslateSet(uni, v))
	return r
}
