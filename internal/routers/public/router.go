package public

import (
	publicui "blog-server/internal/public/ui"
	"blog-server/internal/routers"
	v1 "blog-server/internal/routers/api/v1"
	"blog-server/internal/routers/ping"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	r := routers.NewEngine()
	article := v1.NewArticle()
	tags := v1.NewTag()
	pingHandler := ping.NewPing()

	publicHTML, err := publicui.Files.ReadFile("assets/public.html")
	if err != nil {
		panic(err)
	}
	staticFiles, err := fs.Sub(publicui.Files, "assets")
	if err != nil {
		panic(err)
	}
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", publicHTML)
	})
	r.StaticFS("/assets", http.FS(staticFiles))
	r.GET("/test/ping", pingHandler.Pong)

	api := r.Group("/api/v1/public")
	api.GET("/articles", article.PublicList)
	api.GET("/articles/:id", article.PublicGet)
	api.GET("/tags", tags.PublicList)
	return r
}
