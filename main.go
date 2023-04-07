package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type App struct {
	compressor Compressor
	filer      Filer
	router     *gin.RouterGroup
}

func NewApp(compressor Compressor, filerType string, router *gin.RouterGroup) *App {
	var filer Filer
	var err error
	switch filerType {
	case FS_BADGER:
		filer, err = NewBFS(&FileStorage{compressor})
		if err != nil {
			panic(err)
		}
		break
	case FS_IO:
		filer = NewIOFileStorage(&FileStorage{compressor})
		break
	}
	return &App{
		compressor,
		filer,
		router,
	}
}

func (a *App) Cleanup() {
	defer a.filer.Cleanup()
}

func (app *App) handleFileGetById(ctx *gin.Context) {
	// Pitäiskö gzhttp wrapper ottaa käyttöön händlää end to end compressio?
	// gz, err := gzhttp.NewWrapper(gzhttp.MinSize(1000), gzhttp.CompressionLevel(gzip.BestSpeed))
	id, err := GetUriId(ctx)
	if err != nil {
		ISE(ctx)
		return
	}
	file, err := app.filer.ReadFile(id)
	if err != nil {
		fmt.Println(err)
		ISE(ctx)
		return
	}
	data := file.Data.Bytes()
	ctx.Header("Content-Disposition", "attachement; filename="+file.Name)
	ctx.Data(http.StatusOK, http.DetectContentType(data[:512]), data)
}

func (app *App) handleFilePut(ctx *gin.Context) {
	fileName := ctx.Request.Header["Key"][0]
	// TODO: Implement content-encoding (decoding) https://www.rfc-editor.org/rfc/rfc9110.html#name-content-encoding
	data, err := ctx.GetRawData()
	if err != nil {
		fmt.Println(err)
		ISE(ctx)
		return
	}
	if f, err := app.filer.SaveFile(fileName, &data); err == nil {
		ctx.JSON(200, f)
	} else {
		fmt.Println(err)
		ISE(ctx)
	}
}

func (app *App) InitRoutes() {
	app.router.GET("/file/:id", app.handleFileGetById)
	app.router.PUT("/file", app.handleFilePut)
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
	root := router.Group("/api")
	badgerRouter := root.Group("/badger")
	app := NewApp(&zlibCompressor{}, FS_BADGER, badgerRouter)
	app.InitRoutes()
	defer app.Cleanup()
	fileRouter := root.Group("/")
	fileApp := NewApp(&zlibCompressor{}, FS_IO, fileRouter)
	fileApp.InitRoutes()
	defer fileApp.Cleanup()

	port := ":8000"
	startupMessage := "===> Starting app"
	startupMessage = startupMessage + " on port " + port
	log.Println(startupMessage)
	router.Run(port)
}
