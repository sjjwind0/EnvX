package socket

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

type frame struct {
	bodySize int     // sizeof(flag) + size(body)
	flag     byte    // requestType
	body     *[]byte // content
	pos      int
}

type stsFrame struct {
	*frame
	sid int
}

func marshalFrame(f *frame, key string) []byte {
	if f.flag == requestType_Hello || f.flag == requestType_ReplyHello ||
		f.flag == requestType_AuthKey {
		return marshalFrameWithoutEncrypt(f, key)
	}
	return marshalFrameWithEncrypt(f, key)
}

func translateToStsFrame(f *frame) *stsFrame {
	stsFrame := new(stsFrame)
	stsFrame.frame = f

	if stsFrame.body != nil {
		stsBody := *f.body
		var sid int = (int(stsBody[0]) << 24) | (int(stsBody[1]) << 16) | (int(stsBody[2]) << 8) | int(stsBody[3])
		stsFrame.pos = 4
		stsFrame.sid = sid
	}
	return stsFrame
}

func marshalFrameWithoutEncrypt(f *frame, key string) []byte {
	var outBytes bytes.Buffer

	if f.body != nil {
		compressData := util.ZlibCompress(f.body)

		compressLength := len(compressData) + 1
		lengthArray := []byte{
			byte(compressLength >> 24),
			byte((compressLength >> 16) & 0xFF),
			byte((compressLength >> 8) & 0xFF),
			byte(compressLength & 0xFF),
		}
		outBytes.Write(lengthArray)
		outBytes.WriteByte(byte(f.flag))
		outBytes.Write(compressData)
	} else {
		outBytes.Write([]byte{0, 0, 0, 1})
		outBytes.WriteByte(byte(f.flag))
	}

	return outBytes.Bytes()
}

func marshalFrameWithEncrypt(f *frame, key string) []byte {
	var outBytes bytes.Buffer
	if f.body != nil {
		out := util.AESEncrypt(key, *f.body)
		compressData := util.ZlibCompress(&out)

		compressLength := len(compressData) + 1
		lengthArray := []byte{
			byte(compressLength >> 24),
			byte((compressLength >> 16) & 0xFF),
			byte((compressLength >> 8) & 0xFF),
			byte(compressLength & 0xFF),
		}
		outBytes.Write(lengthArray)
		// outBytes.WriteByte(byte(1 + len(compressData)))
		outBytes.WriteByte(byte(f.flag))
		outBytes.Write(compressData)
	} else {
		outBytes.Write([]byte{0, 0, 0, 1})
		outBytes.WriteByte(byte(f.flag))
	}

	return outBytes.Bytes()
}

func unmarshalFrame(bodySize int, body *[]byte, key []byte) (*frame, error) {
	f := new(frame)
	f.flag = (*body)[0]
	frameBody := (*body)[1:]

	var encryptData []byte = nil
	if bodySize > 1 {
		var err error
		encryptData, err = util.ZlibUnCompress(&frameBody)
		if err != nil {
			return nil, err
		}
	}

	if f.flag == requestType_Hello || f.flag == requestType_ReplyHello || f.flag == requestType_AuthKey {
		f.bodySize = bodySize
		if encryptData != nil {
			f.body = &encryptData
		}
	} else {
		if encryptData != nil {
			decryptedData, err := util.AESDecrypt(key, encryptData)
			if err != nil {
				return nil, err
			}
			f.body = &decryptedData
		}
		f.bodySize = bodySize
	}
	return f, nil
}

func newHelloBuffer() []byte {
	f := new(frame)
	f.flag = requestType_Hello
	f.body = nil

	return marshalFrame(f, "")
}

func newReplyBuffer(publicRSAKey []byte) []byte {
	f := new(frame)
	f.flag = requestType_ReplyHello
	f.body = &publicRSAKey

	return marshalFrame(f, "")
}

func newAuthkeyBuffer(publicRSAKey []byte, aesKey []byte) []byte {
	f := new(frame)
	f.flag = requestType_AuthKey
	encryptData, err := util.RSAPublicKeyEncrypt(&publicRSAKey, &aesKey)
	if err != nil {
		return nil
	}
	f.body = &encryptData

	return marshalFrame(f, "")
}

func newNewRequestBuffer(key string) []byte {
	f := new(frame)
	f.flag = reqeustType_NewRequest
	f.body = nil
	return marshalFrame(f, key)
}

func newPingBuffer(key string) []byte {
	f := new(frame)
	f.flag = requestType_Ping
	f.body = nil
	return marshalFrame(f, key)
}

func newBodyFromSid(sid int, body *[]byte) *[]byte {
	var bodyBuffer bytes.Buffer
	sidArray := []byte{
		byte(sid >> 24),
		byte((sid >> 16) & 0xFF),
		byte((sid >> 8) & 0xFF),
		byte(sid & 0xFF),
	}
	bodyBuffer.Write(sidArray)
	if body != nil {
		bodyBuffer.Write(*body)
	}
	var bodyBytes []byte = make([]byte, bodyBuffer.Len())
	copy(bodyBytes, bodyBuffer.Bytes())
	return &bodyBytes
}

func newContentBuffer(sid int, body *[]byte, key string) []byte {
	f := new(frame)
	f.flag = reqeustType_Content
	f.body = newBodyFromSid(sid, body)
	return marshalFrame(f, key)
}

func newCloseBuffer(sid int, key string) []byte {
	f := new(frame)
	f.flag = requestType_Close
	f.body = newBodyFromSid(sid, nil)
	return marshalFrame(f, key)
}

func newErrorBuffer(sid int, msg string, key string) []byte {
	f := new(frame)
	f.flag = requestType_Error
	msgBytes := []byte(msg)
	f.body = newBodyFromSid(sid, &msgBytes)
	return marshalFrame(f, key)
}
