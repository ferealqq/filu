package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadgerSaveAndRead(t *testing.T) {
	var bfs, err = NewBFS(&FileStorage{compressor: &zlibCompressor{}})
	assert.Nil(t, err)

	defer bfs.Cleanup()

	bs, err := readFile("./test.png")
	assert.Nil(t, err)
	ftx, err := bfs.SaveFile("test.png", &bs)
	assert.Nil(t, err)
	assert.Equal(t, ftx.ContentType, "image/png")

	file, err := bfs.ReadFile(ftx.Id)

	assert.Nil(t, err)

	res := file.Data.Bytes()

	assert.Equal(t, len(res), len(bs))
	assert.Equal(t, http.DetectContentType(res[:512]), http.DetectContentType(bs[:512]))
}
