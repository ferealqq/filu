package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type App struct {
	compressor Compressor
	db         *gorm.DB
	filer      Filer
}

func NewApp(compressor Compressor, filerType string) *App {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&FileBlock{})
	var filer Filer
	switch filerType {
	case FS_BADGER:
		filer, err = NewBFS(&FileStorage{compressor})
		if err != nil {
			panic(err)
		}
		break
	case FS_IO:
		filer = &IOFileStorage{&FileStorage{compressor}}
		break
	}
	return &App{
		compressor,
		db,
		filer,
	}
}

func (a *App) Cleanup() {
	defer a.filer.Cleanup()
}

type UriId struct {
	// TODO injection proof
	ID string `uri:"id" binding:"required"`
}

// Response is a custom response object we pass around the system and send back to the customer
// 404: Not found
// 500: Internal Server Error
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func GetUriId(ctx *gin.Context) (string, error) {
	var uri UriId
	if e := ctx.ShouldBindUri(&uri); e != nil {
		ctx.JSON(http.StatusBadRequest, Response{
			Status:  strconv.Itoa(http.StatusBadRequest),
			Message: "malformed id",
		})
		return "", e
	}
	return uri.ID, nil
}

func ISE(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, Response{
		Status:  strconv.Itoa(http.StatusInternalServerError),
		Message: "Something went wrong",
	})
}

const testFileName = "test.png"

func main() {
	router := gin.Default()
	app := NewApp(&zlibCompressor{}, FS_BADGER)
	defer app.Cleanup()

	router.GET("/file", func(ctx *gin.Context) {
		ctx.Header("Content-Disposition", "attachement; filename="+testFileName)
		f := &IOFileStorage{&FileStorage{app.compressor}}
		file, err := f.ReadFile("./test.zlib")
		// contentType, _, _ := downloadFile("./test.png")
		if err != nil {
			fmt.Println("error")
			return
		}
		data := file.Data.Bytes()
		ctx.Data(http.StatusOK, http.DetectContentType(data[:512]), data)
	})

	router.GET("/badger/file/:id", func(ctx *gin.Context) {
		id, err := GetUriId(ctx)
		if err != nil {
			return
		}
		file, err := app.filer.ReadFile(id)
		if err != nil {
			return
		}
		data := file.Data.Bytes()
		ctx.Header("Content-Disposition", "attachement; filename="+file.Name)
		ctx.Data(http.StatusOK, http.DetectContentType(data[:512]), data)
	})

	router.PUT("/badger/file", func(ctx *gin.Context) {
		fileName := ctx.Request.Header["Key"][0]
		// TODO: Implement content-encoding (decoding) https://www.rfc-editor.org/rfc/rfc9110.html#name-content-encoding
		data, err := ctx.GetRawData()
		if err != nil {
			ISE(ctx)
			return
		}
		if f := app.filer.SaveFile(fileName, &data); f.Error == nil {
			ctx.JSON(200, f)
		} else {
			ISE(ctx)
		}
	})
	port := ":8000"
	router.Run(port)

	fmt.Printf("Hosting on port %s \n", port)
}
