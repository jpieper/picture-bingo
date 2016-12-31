package picture_bingo

import ("image"
	"log"
	"net/http"
	"image/jpeg"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"google.golang.org/appengine"
	_ "image/png"
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
		c.JSON(200, gin.H{"pitctures": []gin.H{
			gin.H{"name": "hello",
				"url": "hello_url"},
			gin.H{"name": "biz",
				"url": "bizzy_url"}}})
	})

	router.POST("/v1/add_picture", func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		filename := header.Filename
		if (err != nil) {
			log.Print(err)
			return
		}
		img, _, err := image.Decode(file)
		if (err != nil) {
			log.Print(err)
			return
		}
		log.Printf("copied: %s", filename)
		small := resize.Thumbnail(320, 320, img, resize.Bicubic)
		log.Print(small.Bounds())

		appengine_context := appengine.NewContext(c.Request)
		client, err := storage.NewClient(appengine_context)
		if (err != nil) {
			log.Print(err)
			return
		}


		bucket := client.Bucket("picture-bingo.appspot.com")

		// TODO: Make object be in directory named for card
		// with unique id for image.
		object := bucket.Object("test_data")

		object_writer := object.NewWriter(appengine_context)

		jpeg.Encode(object_writer, small, nil)
		err = object_writer.Close()
		if (err != nil) {
			log.Print(err)
			return
		}

		// TODO: Try to add it to the list for this card.
		// Remove object if this fails.

	})

	http.Handle("/", router)
}
