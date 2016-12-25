package util

import (
	"bytes"
	"compress/zlib"
	"io"
)

func ZlibCompress(data *[]byte) []byte {
	var compress bytes.Buffer
	writer := zlib.NewWriter(&compress)
	writer.Write(*data)
	writer.Close()

	return compress.Bytes()
}

func ZlibUnCompress(data *[]byte) ([]byte, error) {
	dataReader := bytes.NewReader(*data)
	reader, err := zlib.NewReader(dataReader)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	io.Copy(&out, reader)
	reader.Close()
	return out.Bytes(), nil
}
