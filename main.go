package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	Code  string
	Price uint
}

type FileBlock struct {
	gorm.Model
	Index    int    `gorm:"default:null"`
	Data     []byte `gorm:"default:null"`
	Type     string // "meta" => includes all the metadata, "block" => contains the bytes
	FileType string `gorm:"default:null"`
	FileName string `gorm:"default:null"`
}

var zs = new(zsCompressor)
var zl = new(zlibCompressor)

const FileReadWrite = 0666

func downloadFile(filePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type App struct {
	compressor Compressor
	db         *gorm.DB
	bb         *badger.DB
}

func NewApp(compressor Compressor) *App {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&FileBlock{})
	bb, err := badger.Open(badger.DefaultOptions("badger_db"))
	return &App{
		compressor,
		db,
		bb,
	}
}

func (app *App) downloadCompressedFile(filePath string) (string, []byte, error) {
	compressed, err := downloadFile(filePath)
	if err != nil {
		return "", nil, err
	}
	if bytes, err := app.compressor.Decompress(compressed); err == nil {
		content := http.DetectContentType(bytes)
		return content, bytes, nil
	} else {
		return "", nil, err
	}
}

func (app *App) saveFile(fileName string, data []byte) error {
	compressed := app.compressor.Compress(data)
	index := 0
	if err := app.db.Create(&FileBlock{
		Index:    index,
		Type:     "meta",
		FileName: fileName,
		FileType: http.DetectContentType(data),
	}).Error; err != nil {
		return err
	}

	app.saveFileParts(1+index, compressed)

	return nil
}

func (app *App) bSaveFile(fileName string, data []byte) error {
	compressed := app.compressor.Compress(data)
	err := app.bb.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(fileName), compressed)
		return err
	})

	return err
}

func (app *App) saveFileParts(index int, data []byte) error {
	// FIXME: the size of the blocks should be calculated according to the fullsize of the file.
	size := len(data)
	part := 128000
	next := part
	for i := 0; i < size; i += part {
		next = i + part
		var block []byte
		if next > size {
			block = data[i:size]
		} else {
			block = data[i:next]
		}
		index = index + 1
		app.db.Create(&FileBlock{
			Index: index,
			Data:  block,
			Type:  "block",
		})
	}
	return nil
}

type FileContext struct {
	FileName    string
	Data        []byte
	Error       error
	ContentType string
}

func (app *App) sqlite() *FileContext {
	var blocks []FileBlock
	meta := &FileBlock{}
	if err := app.db.First(meta, &FileBlock{Type: "meta"}).Error; err != nil {
		return &FileContext{Error: err}
	}
	if err := app.db.Find(&blocks, &FileBlock{Type: "block"}).Error; err != nil {
		return &FileContext{Error: err}
	}
	var fileBuffer bytes.Buffer
	writer := bufio.NewWriter(&fileBuffer)
	for _, block := range blocks {
		if _, err := writer.Write(block.Data); err != nil {
			return &FileContext{Error: err}
		}
	}
	decompressed, err := app.compressor.Decompress(fileBuffer.Bytes())
	if err != nil {
		return &FileContext{Error: err}
	}
	return &FileContext{
		Data:        decompressed,
		FileName:    meta.FileName,
		ContentType: http.DetectContentType(decompressed),
	}
}

func (app *App) badger() *FileContext {
	return nil
}

const testFileName = "test.png"

func _main() {
	data, _ := ioutil.ReadFile("./" + testFileName)
	// // reader := bytes.NewReader(data)
	// bs := zl.Compress(data)
	app := NewApp(&zlibCompressor{})
	app.bSaveFile(testFileName, data)
	// fmt.Println(bs)
	// os.WriteFile("test.go.zlib", bs, FileReadWrite)
}

func main() {
	router := gin.Default()
	app := NewApp(&zlibCompressor{})
	defer app.bb.Close()

	router.GET("/file", func(ctx *gin.Context) {
		ctx.Header("Content-Disposition", "attachement; filename="+testFileName)
		contentType, data, err := app.downloadCompressedFile("./test.zlib")
		// contentType, _, _ := downloadFile("./test.png")
		if err != nil {
			fmt.Println("error")
			return
		}
		fmt.Println(contentType)
		ctx.Data(http.StatusOK, contentType, data)
	})

	router.GET("/sql/file", func(ctx *gin.Context) {
		fileContext := app.sqlite()
		if fileContext.Error != nil {
			fmt.Println("error")
			return
		}
		ctx.Header("Content-Disposition", "attachement; filename="+fileContext.FileName)
		ctx.Data(http.StatusOK, fileContext.ContentType, fileContext.Data)
	})

	router.GET("/badger/file", func(ctx *gin.Context) {
		err := app.bb.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(testFileName))
			if err != nil {
				return err
			}
			err = item.Value(func(val []byte) error {
				decompress, _ := zl.DecompressPointer(&val)
				ctx.Header("Content-Disposition", "attachement; filename="+testFileName)
				ctx.Data(http.StatusOK, http.DetectContentType(decompress), decompress)

				return nil
			})
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	})
	port := ":8000"
	router.Run(port)

	fmt.Printf("Hosting on port %s \n", port)
}
