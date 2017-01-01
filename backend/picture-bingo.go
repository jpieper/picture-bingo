package picture_bingo

import ("image"
	"log"
	"math/big"
	"net/http"
	"net/url"
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
	_ "image/png"
)

type Picture struct {
	cloud_id string
	web_url url.URL
}

type Card struct {
	pictures []Picture
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

	object := bucket.Object(name)

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
	encoder.Encode(card)

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
		_ = write_card(appengine_context, bucket, name, new_card, ver)
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

	object := bucket.Object(pictureUuid)
	object_writer := object.NewWriter(appengine_context)

	jpeg.Encode(object_writer, small, nil)
	err = object_writer.Close()
	if (err != nil) {
		panic(err)
	}

	blob_key, err := blobstore.BlobKeyForFile(
		appengine_context, "/gs/picture-bingo.appspot.com/" + pictureUuid)
	if (err != nil) {
		panic(err)
	}

	url, err := aimage.ServingURL(appengine_context, blob_key, nil)
	if (err != nil) {
		panic(err)
	}

	update_card(appengine_context, cardName, func(card Card) Card {
		pic := Picture{cloud_id: pictureUuid, web_url: *url}
		card.pictures = append(card.pictures, pic)
		// TODO: Fail in some way if we have too many pictures.
		return card
	})
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
		err := write_card(appengine_context, bucket, newCardName, card, 0)
		if (err != nil) {
			panic(err)
		}
		c.JSON(200, gin.H{"name": newCardName})
	})

	router.GET("/v1/get_card/:name", func(c *gin.Context) {
		c.JSON(200, gin.H{"pictures": []gin.H{
			gin.H{"name": "hello",
				"url": "hello_url"},
			gin.H{"name": "biz",
				"url": "bizzy_url"}}})
	})

	router.POST("/v1/add_picture/:name", add_picture)
	http.Handle("/", router)
}
