package conn

import (
	"testing"
)

func TestFrameMarshalAndUnMarshal_Hello(t *testing.T) {
	hello := newHelloBuffer()
	_, f, err := unmarshalFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Hello {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Reply(t *testing.T) {
	hello := newReplyBuffer()
	_, f, err := unmarshalFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_ReplyHello {
		t.Error("error")
	}
	if string(*f.body) != string(currentPublicKey) {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Auth(t *testing.T) {
	hello := newAuthkeyBuffer([]byte("1234567890123456"))
	_, f, err := unmarshalFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_AuthKey {
		t.Error("error")
	}

	if string(*f.body) != "1234567890123456" {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Content(t *testing.T) {
	data := []byte("1234567890123456")
	key := "1234567890123456"
	hello := newContentBuffer(&data, key)
	_, f, err := unmarshalFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != reqeustType_Content {
		t.Error("error")
	}

	if string(*f.body) != string(data) {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Ping(t *testing.T) {
	key := "1234567890123456"
	hello := newPingBuffer(key)
	_, f, err := unmarshalFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Ping {
		t.Error("error")
	}
}
