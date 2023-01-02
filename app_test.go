package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadComperssedFile(t *testing.T) {
	app := NewApp(&zlibCompressor{}, FS_IO)
	defer app.Cleanup()
	file, err := app.filer.ReadFile("test.zlib")
	assert.NotNil(t, file)
	assert.Nil(t, err)
	assert.Equal(t, len(file.Data.Bytes()), 1261800)
	assert.Equal(t, "image/png", http.DetectContentType(file.Data.Bytes()[:512]))
}
