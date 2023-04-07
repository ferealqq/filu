package main

import (
	"bufio"
	"bytes"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
)

type Compressor interface {
	Compress(src *[]byte, dst *bytes.Buffer) (int, error)
	Decompress(src *bytes.Buffer, dst *bytes.Buffer) (int64, error)
	FileExtension() string
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

func (_ *zsCompressor) FileExtension() string {
	return ".zst"
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

func (_ *zlibCompressor) FileExtension() string {
	return ".zlib"
}

type s2Compressor struct{}

func (_ *s2Compressor) Compress(src *[]byte, dst *bytes.Buffer) (int, error) {
	enc := s2.NewWriter(dst)
	// The encoder owns the buffer until Flush or Close is called.
	err := enc.EncodeBuffer(*src)
	if err != nil {
		enc.Close()
		return err
	}
	// Blocks until compression is done.
	return enc.Close()
	w := s2.NewWriter(dst)
	defer w.Close()
	return w.Write(*src)
}

func (_ *s2Compressor) Decompress(src *bytes.Buffer, dst *bytes.Buffer) (int64, error) {
	r, err := s2.NewReader(src)
	if err != nil {
		return 0, err
	}

	writer := bufio.NewWriter(dst)
	return writer.ReadFrom(r)
}

func (_ *s2Compressor) FileExtension() string {
	return ".s2"
}
