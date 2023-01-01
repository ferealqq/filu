package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadSQLFile(t *testing.T) {
	app := NewApp(&zlibCompressor{})
	res := app.sqlite()
	assert.NotNil(t, res)
	assert.Nil(t, res.Error)
	assert.Equal(t, len(res.Data), 1261800)
	assert.Equal(t, "image/png", res.ContentType)
}
