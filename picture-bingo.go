package picture_bingo

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func init() {
    router := gin.New()
    router.Use(gin.Logger())
    router.LoadHTMLGlob("templates/*.tmpl.html")
    router.Static("/static", "static")

    router.GET("/", func(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl.html", nil)
    })

    http.Handle("/", router)
}
