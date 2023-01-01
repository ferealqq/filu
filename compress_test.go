package main

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZlibDecompress(t *testing.T) {
	c := &zlibCompressor{}
	bs, _ := readFile("./test.zlib")
	var src = bytes.NewBuffer(bs)
	var dst bytes.Buffer
	_, err := c.Decompress(src, &dst)
	assert.Nil(t, err)
	res := dst.Bytes()
	assert.Equal(t, len(res), 1261800)
	assert.Equal(t, "image/png", http.DetectContentType(res[:512]))
}
