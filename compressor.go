package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"

	"github.com/klauspost/compress/zstd"
)

var encoder, _ = zstd.NewWriter(nil)

type Compressor interface {
	Compress(src []byte) []byte
	Decompress(src []byte) ([]byte, error)
}

type zsCompressor struct{}

// Compress a buffer.
// If you have a destination buffer, the allocation in the call can also be eliminated.
func (_ *zsCompressor) Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func (_ *zsCompressor) Decompress(src []byte) ([]byte, error) {
	in := bytes.NewBuffer(src)
	r, err := zstd.NewReader(in)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var bs bytes.Buffer
	writer := bufio.NewWriter(&bs)
	if _, err = io.Copy(writer, r); err != nil {
		return nil, err
	}
	return bs.Bytes(), nil
}

type zlibCompressor struct{}

func (_ *zlibCompressor) Compress(src []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(src)
	w.Close()
	return b.Bytes()
}

func (_ *zlibCompressor) Decompress(src []byte) ([]byte, error) {
	b := bytes.NewBuffer(src)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}

	var bs bytes.Buffer
	writer := bufio.NewWriter(&bs)
	_, errs := io.Copy(writer, r)
	if errs != nil {
		return nil, err
	}
	r.Close()
	return bs.Bytes(), nil
}

func (_ *zlibCompressor) DecompressPointer(src *[]byte) ([]byte, error) {
	b := bytes.NewBuffer(*src)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}

	var bs bytes.Buffer
	writer := bufio.NewWriter(&bs)
	_, errs := io.Copy(writer, r)
	if errs != nil {
		return nil, err
	}
	r.Close()
	return bs.Bytes(), nil
}
