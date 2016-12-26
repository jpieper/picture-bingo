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

	router.GET("/v1/make_new_card", func(c *gin.Context) {
		c.JSON(200, gin.H{"name": "stuff"})
	})

	router.GET("/v1/get_card", func(c *gin.Context) {
		pictures := []string{"hello", "stuff"}
		c.JSON(200, gin.H{"pictures": pictures})
	})

	http.Handle("/", router)
}
