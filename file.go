package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type FileBlock struct {
	// TODO: fix this mess if we want to use sqlite, this was just for prototyping
	gorm.Model
	Type     string    `gorm:"default:null"`
	Name     string    `gorm:"default:null"`
	Id       uuid.UUID `gorm:"type:uuid;primary_key;"`
	Location string    `gorm:"default:null"`
}
type FileContext struct {
	Name        string
	ContentType string
	Id          string
}

type File struct {
	// does not contain contentType, because we don't want to do any unnecessary prosessing of the file bytes
	// ContentType should be retrieved from Data.Bytes() in the http request handler
	Name string
	Id   string
	Data *bytes.Buffer
}

type Filer interface {
	SaveFile(fileName string, data *[]byte) (*FileContext, error)
	ReadFile(fileId string) (*File, error)
	Cleanup() error
}

const (
	// filers
	FS_BADGER = "FILER_BADGER"
	FS_IO     = "FILER_IO"
)

// universally used fileStorage struct to keep hold of variables that are shared between FileStorage implementations such as BadgerFileStorage
type FileStorage struct {
	compressor Compressor
}

var zs = new(zsCompressor)
var zl = new(zlibCompressor)

const FileReadWrite = 0666

func readFile(filePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// this key represents the bytes of a file in the badger database
func dataKey(fileId string) []byte {
	return []byte("file:///" + fileId)
}

// Badger file storage service
type BadgerFileStorage struct {
	*FileStorage
	db *badger.DB
}

// create new BadgerFileStorage
func NewBFS(fs *FileStorage) (*BadgerFileStorage, error) {
	// TODO add badger to config
	db, err := badger.Open(badger.DefaultOptions("badger_db"))
	if err != nil {
		return nil, err
	}
	return &BadgerFileStorage{
		fs,
		db,
	}, nil
}

func (b BadgerFileStorage) SaveFile(fileName string, data *[]byte) (*FileContext, error) {
	// Badger value limit is 1mb by default
	// FIXME: https://github.com/dgraph-io/badger/issues/60
	var compressed bytes.Buffer
	_, err := b.compressor.Compress(data, &compressed)
	if err != nil {
		return nil, err
	}
	id := uuid.New().String()
	err = b.db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(id), []byte(fileName))
		err = txn.Set(dataKey(id), compressed.Bytes())
		return err
	})

	if err != nil {
		return nil, err
	}

	return &FileContext{
		ContentType: http.DetectContentType((*data)[:512]),
		Name:        fileName,
		Id:          id,
	}, nil
}

func (b BadgerFileStorage) Cleanup() error {
	return b.db.Close()
}

func (b BadgerFileStorage) ReadFile(fileId string) (*File, error) {
	var decompress = new(bytes.Buffer)
	var fileName string
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(fileId))
		if err != nil {
			return err
		}
		bsName, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		fileName = string(bsName)
		item, err = txn.Get(dataKey(fileId))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			// https://dgraph.io/docs/badger/get-started/#using-key-value-pairs
			// Don't use the val outside of this function. val should only be copied out of this function but never reassigned
			var valBuf bytes.Buffer
			if _, err := valBuf.Write(val); err != nil {
				return err
			}
			_, err := zl.Decompress(&valBuf, decompress)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &File{
		Name: fileName,
		Id:   fileId,
		Data: decompress,
	}, nil
}

type IOFileStorage struct {
	*FileStorage
	db *gorm.DB
}

func NewIOFileStorage(fileStorage *FileStorage) *IOFileStorage {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&FileBlock{})
	return &IOFileStorage{
		fileStorage,
		db,
	}
}

func (f IOFileStorage) ReadFile(fileId string) (*File, error) {
	fileBlock := new(FileBlock)
	if err := f.db.Where("Id = ?", fileId).First(&fileBlock).Error; err != nil {
		return nil, err
	}
	bs, err := readFile(fileBlock.Location)
	if err != nil {
		return nil, err
	}
	var decompressed = new(bytes.Buffer)
	_, err = f.compressor.Decompress(bytes.NewBuffer(bs), decompressed)
	if err != nil {
		return nil, err
	}
	return &File{
		Name: fileBlock.Name,
		Data: decompressed,
		Id:   fileId,
	}, nil
}

func (f IOFileStorage) SaveFile(fileName string, data *[]byte) (*FileContext, error) {
	id := uuid.New()
	compressed := new(bytes.Buffer)
	_, err := f.compressor.Compress(data, compressed)
	if err != nil {
		return nil, err
	}
	// TODO: absolute path from config
	cur, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := cur + "/files/" + id.String() + f.compressor.FileExtension()
	err = os.WriteFile(path, compressed.Bytes(), FileReadWrite)
	if err != nil {
		return nil, err
	}
	fileBlock := &FileBlock{
		Id:       id,
		Name:     fileName,
		Location: path,
		Type:     http.DetectContentType((*data)[:512]),
	}
	err = f.db.Create(fileBlock).Error

	return &FileContext{
		Name:        fileName,
		ContentType: fileBlock.Type,
		Id:          id.String(),
	}, err
}

func (f IOFileStorage) Cleanup() error {
	return nil
}
