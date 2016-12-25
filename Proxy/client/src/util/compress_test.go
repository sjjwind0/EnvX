package util

import "testing"

func TestCompressAndUnCompress(t *testing.T) {
	data := []byte("Hello World")
	out := ZlibCompress(&data)
	readData, err := ZlibUnCompress(&out)
	if err != nil {
		t.Error(err)
	}
	if string(readData) != string(data) {
		t.Error("not match")
	}
}
