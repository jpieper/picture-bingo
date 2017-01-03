package picture_bingo

import ("image"
	"log"
	"math/big"
	"net/http"
	"image/jpeg"
	"encoding/json"
	"strconv"
	"crypto/rand"
	"golang.org/x/net/context"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"github.com/google/uuid"
	"google.golang.org/appengine"
	"google.golang.org/appengine/blobstore"
	aimage "google.golang.org/appengine/image"
	"github.com/jung-kurt/gofpdf"
	_ "image/png"
)

type Picture struct {
	CloudID string `json:"cloud_id"`
	WebURL string `json:"web_url"`
}

type Card struct {
	Pictures []Picture `json:"pictures"`
}

func get_client(appengine_context context.Context) (*storage.Client) {
	client, err := storage.NewClient(appengine_context)
	if (err != nil) {
		panic(err);
	}

	return client
}

func get_bucket(client *storage.Client) (*storage.BucketHandle) {
	return client.Bucket("picture-bingo.appspot.com")
}

// Return a card and its generation.
func get_card(appengine_context context.Context, bucket *storage.BucketHandle, name string) (Card, int64) {
	object := bucket.Object(name + "/info")
	attrs, err := object.Attrs(appengine_context)
	if (err != nil) {
		panic(err);
	}

	reader, err := object.NewReader(appengine_context)
	if (err != nil) {
		panic(err);
	}

	var card Card

	decoder := json.NewDecoder(reader)
	err = decoder.Decode(&card)
	if (err != nil) {
		panic(err)
	}

	return card, attrs.Generation
}

func write_card(appengine_context context.Context, bucket *storage.BucketHandle,
	name string, card Card, required_generation int64) (error) {

	object := bucket.Object(name + "/info")

	maybe_object := func() *storage.ObjectHandle {
		if (required_generation != 0) {
			return object.If(
				storage.Conditions{
					GenerationMatch: required_generation})
		} else {
			return object
		}
	}();

	writer := maybe_object.NewWriter(appengine_context)
	encoder := json.NewEncoder(writer)
	err := encoder.Encode(&card)
	if (err != nil) {
		panic(err)
	}

	return writer.Close()
}

type CardUpdater func(Card) Card

func update_card(appengine_context context.Context, name string, updater CardUpdater) {
	client := get_client(appengine_context)
	bucket := get_bucket(client)
	for {
		card, ver := get_card(appengine_context, bucket, name)
		new_card := updater(card)
		// TODO: Figure out what error is returned for a
		// precondition failure.
		err := write_card(appengine_context, bucket, name, new_card, ver)
		if (err != nil) {
			panic(err)
		}
		break
	}
}

func add_picture(c *gin.Context) {
	cardName := c.Param("name")

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
	client := get_client(appengine_context)
	bucket := get_bucket(client)

	pictureUuid := uuid.New().String()

	object := bucket.Object(cardName + "/" + pictureUuid)
	object_writer := object.NewWriter(appengine_context)

	jpeg.Encode(object_writer, small, nil)
	err = object_writer.Close()
	if (err != nil) {
		panic(err)
	}

	blob_key, err := blobstore.BlobKeyForFile(
		appengine_context, "/gs/picture-bingo.appspot.com/" + cardName + "/" + pictureUuid)
	if (err != nil) {
		panic(err)
	}

	url, err := aimage.ServingURL(appengine_context, blob_key, nil)
	if (err != nil) {
		panic(err)
	}

	update_card(appengine_context, cardName, func(card Card) Card {
		pic := Picture{CloudID: pictureUuid, WebURL: url.String()}
		card.Pictures = append(card.Pictures, pic)
		// TODO: Fail in some way if we have too many pictures.
		return card
	})

	c.JSON(200, gin.H{"status": "OK"})
}

func getRandInt(max int) int {
	result, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if (err != nil) {
		panic(err)
	}
	return int(result.Int64())
}

func getRandomName() string {
	return adjectiveList[getRandInt(len(adjectiveList))] + "-" + nounList[getRandInt(len(nounList))] + "-" + strconv.Itoa(getRandInt(10000))
}

func init() {
	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.GET("/v1/make_new_card", func(c *gin.Context) {
		newCardName := getRandomName()

		appengine_context := appengine.NewContext(c.Request)
		client := get_client(appengine_context)
		bucket := get_bucket(client)
		var card Card
		card.Pictures = make([]Picture, 0)
		err := write_card(appengine_context, bucket, newCardName, card, 0)
		if (err != nil) {
			panic(err)
		}
		c.JSON(200, gin.H{"name": newCardName})
	})

	router.GET("/v1/get_card/:name", func(c *gin.Context) {
		appengine_context := appengine.NewContext(c.Request)
		client := get_client(appengine_context)
		bucket := get_bucket(client)
		card, _ := get_card(appengine_context, bucket, c.Param("name"))

		c.JSON(200, card)
	})

	router.GET("/v1/make_pdf/:name", func(c *gin.Context) {
		pdf := gofpdf.New("P", "mm", "letter", "")
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 16)
		pdf.Cell(40, 10, "Hello world!")
		c.Status(200)
		pdf.Output(c.Writer)
	})

	router.POST("/v1/add_picture/:name", add_picture)
	http.Handle("/", router)
}
