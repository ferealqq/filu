package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZlibDecompress(t *testing.T) {
	c := &zlibCompressor{}
	bs, _ := downloadFile("./test.zlib")
	res, _ := c.Decompress(bs)

	assert.Equal(t, len(res), 1261800)
	assert.Equal(t, "image/png", http.DetectContentType(res))
}
