package main

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZlibDecompress(t *testing.T) {
	c := &zlibCompressor{}
	bs, _ := readFile("./test_files/test.zlib")
	var src = bytes.NewBuffer(bs)
	var dst bytes.Buffer
	_, err := c.Decompress(src, &dst)
	assert.Nil(t, err)
	res := dst.Bytes()
	assert.Equal(t, len(res), 82569)
	assert.Equal(t, "image/jpeg", http.DetectContentType(res[:512]))
}
