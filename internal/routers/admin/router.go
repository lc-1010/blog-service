package admin

import (
	_ "blog-server/docs"
	"blog-server/internal/admin"
	adminui "blog-server/internal/admin/ui"
	"blog-server/internal/routers"
	v1 "blog-server/internal/routers/api/v1"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New() *gin.Engine {
	r := routers.NewEngine()
	article := v1.NewArticle()
	tags := v1.NewTag()
	adminHandler := admin.NewHandler()
	adminHandler.RegisterRoutes(r)
	registerUI(r)

	api := r.Group("/api/v1")
	api.Use(adminHandler.AuthRequired())
	api.Use(adminHandler.AuditMutations())
	content := api.Group("")
	content.Use(adminHandler.RequirePermission("content.manage"))
	content.POST("/tags", tags.Create)
	content.DELETE("/tags/:id", tags.Delete)
	content.PUT("/tags/:id", tags.Update)
	content.PATCH("/tags/:id/state", tags.Update)
	content.GET("/tags", tags.List)
	content.POST("/articles", article.Create)
	content.GET("/articles", article.List)
	content.GET("/articles/:id", article.Get)
	content.PATCH("/articles/:id/state", article.UpdateState)
	content.PUT("/articles/:id", article.Update)
	content.DELETE("/articles/:id", article.Delete)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

func registerUI(r *gin.Engine) {
	staticFiles, err := fs.Sub(adminui.Files, "assets")
	if err != nil {
		panic(err)
	}
	adminHTML, err := adminui.Files.ReadFile("assets/index.html")
	if err != nil {
		panic(err)
	}
	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/admin/")
	})
	r.GET("/admin/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", adminHTML)
	})
	for _, asset := range []struct {
		name        string
		contentType string
	}{
		{name: "app.js", contentType: "text/javascript; charset=utf-8"},
		{name: "style.css", contentType: "text/css; charset=utf-8"},
	} {
		content, err := fs.ReadFile(staticFiles, asset.name)
		if err != nil {
			panic(err)
		}
		r.GET("/admin/"+asset.name, embeddedAssetHandler(asset.contentType, content))
	}
}

func embeddedAssetHandler(contentType string, content []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, contentType, content)
	}
}
