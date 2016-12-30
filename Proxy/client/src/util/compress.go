package util

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

func ZlibCompress(data *[]byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(*data)
	w.Close()
	out := in.Bytes()
	fmt.Println("out: ", out)
	return out
}

func ZlibUnCompress(data *[]byte) ([]byte, error) {
	fmt.Println("in: ", *data)
	b := bytes.NewReader(*data)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	readSize, err := io.Copy(&out, r)
	if err != nil {
		fmt.Println("copy error: ", err)
		return nil, err
	}
	fmt.Println("readSize: ", readSize)
	return out.Bytes(), nil
}
