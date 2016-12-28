package socket

import (
	"testing"
	"util"
)

func unmarshalTestFrame(buffer *[]byte, key string) (*frame, error) {
	bufferSize := (*buffer)[0]
	body := (*buffer)[1:]
	f, err := unmarshalFrame(int(bufferSize), &body, []byte(key))
	return f, err
}

func TestFrameMarshalAndUnMarshal_Hello(t *testing.T) {
	hello := newHelloBuffer()
	f, err := unmarshalTestFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Hello {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Reply(t *testing.T) {
	pubKey, _, _ := util.NewRsaKey(256)
	hello := newReplyBuffer(pubKey)
	f, err := unmarshalTestFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_ReplyHello {
		t.Error("error")
	}
	if string(*f.body) != string(pubKey) {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Auth(t *testing.T) {
	pubKey, privateKey, _ := util.NewRsaKey(256)
	hello := newAuthkeyBuffer(pubKey, []byte("1234567890123456"))
	f, err := unmarshalTestFrame(&hello, "")
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_AuthKey {
		t.Error("error")
		return
	}

	decryptData, err := util.RsaPrivateKeyDecrypt(&privateKey, f.body)
	if err != nil {
		t.Error(err)
		return
	}
	if string(decryptData) != "1234567890123456" {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Content(t *testing.T) {
	data := []byte("1234567890123456")
	key := "1234567890123456"
	hello := newContentBuffer(&data, key)
	f, err := unmarshalTestFrame(&hello, key)
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
	f, err := unmarshalTestFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Ping {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Close(t *testing.T) {
	key := "1234567890123456"
	hello := newCloseBuffer(key)
	f, err := unmarshalTestFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Close {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_NewRequest(t *testing.T) {
	key := "1234567890123456"
	hello := newNewRequestBuffer(key)
	f, err := unmarshalTestFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != reqeustType_NewRequest {
		t.Error("error")
	}
}

func TestFrameMarshalAndUnMarshal_Error(t *testing.T) {
	key := "1234567890123456"
	errMsg := "error message"
	hello := newErrorBuffer(errMsg, key)
	f, err := unmarshalTestFrame(&hello, key)
	if err != nil {
		t.Error("unmarshal error:", err)
		return
	}
	if f.flag != requestType_Error {
		t.Error("error")
	}

	if string(*f.body) != errMsg {
		t.Error("error")
	}
}
