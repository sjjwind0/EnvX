package conn

import (
	"bytes"
	"util"
)

const (
	// connect
	requestType_Hello      = iota // say hello to server
	requestType_ReplyHello = iota // server reply hello, transfer rsa public key
	requestType_AuthKey    = iota // client send aes key to server

	// body
	reqeustType_NewRequest = iota // reset this channel, preprepare for new request, for reuse
	reqeustType_Content    = iota // protocol body info
	requestType_Error      = iota // error when transfer (body recored error info)
	requestType_Close      = iota // close current protocol, for reuse
	requestType_Ping       = iota // ping, keep alive
)

var currentPrivateKey []byte
var currentPublicKey []byte

type frame struct {
	bodySize byte    // sizeof(flag) + size(body)
	flag     byte    // requestType
	body     *[]byte // content
}

func marshalFrame(f *frame, key string) []byte {
	if f.flag == requestType_Hello || f.flag == requestType_ReplyHello ||
		f.flag == requestType_AuthKey {
		return marshalFrameWithoutEncrypt(f, key)
	}
	return marshalFrameWithEncrypt(f, key)
}

func marshalFrameWithoutEncrypt(f *frame, key string) []byte {
	var outBytes bytes.Buffer

	if f.body != nil {
		compressData := util.ZlibCompress(f.body)
		outBytes.WriteByte(byte(len(compressData) + 1))
		outBytes.WriteByte(byte(f.flag))
		outBytes.Write(compressData)
	} else {
		outBytes.WriteByte(1)
		outBytes.WriteByte(byte(f.flag))
	}

	return outBytes.Bytes()
}

func marshalFrameWithEncrypt(f *frame, key string) []byte {
	var outBytes bytes.Buffer
	if f.body != nil {
		out := util.AESEncrypt(key, *f.body)
		compressData := util.ZlibCompress(&out)

		outBytes.WriteByte(byte(1 + len(compressData)))
		outBytes.WriteByte(byte(f.flag))
		outBytes.Write(compressData)
	} else {
		outBytes.WriteByte(1)
		outBytes.WriteByte(byte(f.flag))
	}

	return outBytes.Bytes()
}

func unmarshalFrame(body *[]byte, key string) (int, *frame, error) {
	f := new(frame)
	frameSize := int((*body)[0])
	f.flag = (*body)[1]

	frameBody := (*body)[2 : frameSize+1]

	var encryptData []byte = nil
	if frameSize > 1 {
		var err error
		encryptData, err = util.ZlibUnCompress(&frameBody)
		if err != nil {
			return 0, nil, err
		}
	}

	if f.flag == requestType_Hello || f.flag == requestType_ReplyHello || f.flag == requestType_AuthKey {
		f.bodySize = byte(frameSize)
		if encryptData != nil {
			f.body = &encryptData
		}
	} else {
		if encryptData != nil {
			decryptedData, err := util.AESDecrypt([]byte(key), encryptData)
			if err != nil {
				return 0, nil, err
			}
			f.body = &decryptedData
		}
		f.bodySize = byte(frameSize)
	}
	return frameSize + 1, f, nil
}

func newHelloBuffer() []byte {
	f := new(frame)
	f.flag = requestType_Hello
	f.body = nil

	return marshalFrame(f, "")
}

func newReplyBuffer() []byte {
	if currentPublicKey == nil && currentPrivateKey == nil {
		currentPublicKey, currentPrivateKey, _ = util.NewRsaKey(1024)
	}

	f := new(frame)
	f.flag = requestType_ReplyHello
	f.body = &currentPublicKey

	return marshalFrame(f, "")
}

func newAuthkeyBuffer(aesKey []byte) []byte {
	f := new(frame)
	f.flag = requestType_AuthKey
	f.body = &aesKey

	return marshalFrame(f, "")
}

func newNewRequestBuffer(key string) []byte {
	f := new(frame)
	f.flag = reqeustType_NewRequest
	f.body = nil
	return marshalFrame(f, key)
}

func newContentBuffer(body *[]byte, key string) []byte {
	f := new(frame)
	f.flag = reqeustType_Content
	f.body = body
	return marshalFrame(f, key)
}

func newErrorBuffer(msg string, key string) []byte {
	f := new(frame)
	f.flag = requestType_Error
	body := []byte(msg)
	f.body = &body
	return marshalFrame(f, key)
}

func newPingBuffer(key string) []byte {
	f := new(frame)
	f.flag = requestType_Ping
	f.body = nil
	return marshalFrame(f, key)
}
