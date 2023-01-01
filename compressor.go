package main

import (
	"bufio"
	"bytes"
	"compress/zlib"

	"github.com/klauspost/compress/zstd"
)

type Compressor interface {
	Compress(src *[]byte, dst *bytes.Buffer) (int, error)
	Decompress(src *bytes.Buffer, dst *bytes.Buffer) (int64, error)
}

type zsCompressor struct{}

// Usage: Compress(src,dst); dst.Bytes()
func (_ *zsCompressor) Compress(src *[]byte, dst *bytes.Buffer) (int, error) {
	writer, err := zstd.NewWriter(dst)
	if err != nil {
		return 0, err
	}
	return writer.Write(*src)
}

func (_ *zsCompressor) Decompress(src *bytes.Buffer, dst *bytes.Buffer) (int64, error) {
	r, err := zstd.NewReader(src)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	return r.WriteTo(dst)
}

type zlibCompressor struct{}

func (_ *zlibCompressor) Compress(src *[]byte, dst *bytes.Buffer) (int, error) {
	w := zlib.NewWriter(dst)
	defer w.Close()
	return w.Write(*src)
}

func (_ *zlibCompressor) Decompress(src *bytes.Buffer, dst *bytes.Buffer) (int64, error) {
	r, err := zlib.NewReader(src)
	if err != nil {
		return 0, err
	}

	writer := bufio.NewWriter(dst)
	return writer.ReadFrom(r)
}
