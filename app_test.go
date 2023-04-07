package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type HttpTestAction struct {
	Name string
	// Functionality of the router
	RouterFunc func(*gin.Engine)
	// Path to which the request will be sent
	ReqPath string
	Method  string
	Body    io.Reader
	Headers http.Header
}

func (action *HttpTestAction) Run() *httptest.ResponseRecorder {
	r, _ := http.NewRequest(action.Method, action.ReqPath, action.Body)
	r.Header = action.Headers
	w := httptest.NewRecorder()
	router := gin.Default()
	// recreate route from routes.go
	action.RouterFunc(router)
	router.ServeHTTP(w, r)

	return w
}

func TestIOFileStorage(t *testing.T) {
	router := gin.Default()
	bs, err := readFile("./test_files/test.jpg")
	assert.Nil(t, err)
	app := NewApp(&zlibCompressor{}, FS_IO, router.Group("/"))
	action := HttpTestAction{
		Method: http.MethodPut,
		RouterFunc: func(e *gin.Engine) {
			e.PUT("/", app.handleFilePut)
		},
		ReqPath: "/",
		Body:    bytes.NewReader(bs),
		Headers: map[string][]string{
			"Key": {
				"test_file_storage.png",
			},
		},
	}
	response := action.Run()
	assert.Equal(t, 200, response.Result().StatusCode)
	var putBody map[string]interface{}

	if err := json.Unmarshal(response.Body.Bytes(), &putBody); err != nil {
		assert.Fail(t, "Unmarshal should not fail")
		return
	}
	assert.Equal(t, "test_file_storage.png", putBody["Name"])
	assert.NotNil(t, putBody["Id"])
	action = HttpTestAction{
		Method: http.MethodGet,
		RouterFunc: func(e *gin.Engine) {
			e.GET("/:id", app.handleFileGetById)
		},
		ReqPath: "/" + putBody["Id"].(string),
	}
	response = action.Run()
	assert.Equal(t, 200, response.Result().StatusCode)
	assert.Equal(t, len(bs), len(response.Body.Bytes()))
	fmt.Println(response.HeaderMap["Content-Disposition"])
}
