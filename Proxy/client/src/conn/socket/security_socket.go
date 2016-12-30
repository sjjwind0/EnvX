package socket

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"util"
)

const (
	kCacheBufferSize = 4 * 1024 // 4KB

	// TODO: read from config file
	kFrameBufferSize = 16 * 1024 // 16KB
)

const (
	status_Init        = iota
	status_Transfering = iota
	status_Closed      = iota
)

var Close = errors.New("Close")

type SecurityTCPSocket struct {
	// about connect
	addr string
	conn net.Conn

	// tcp cache buffer
	cacheBuffer bytes.Buffer

	// tcp real data buffer
	dataBuffer bytes.Buffer

	// about security
	publicRSAKey  []byte
	privateRSAKey []byte
	aesKey        []byte
	frameIndex    int

	status int
}

func NewSecurityClientTCPSocket(addr string) *SecurityTCPSocket {
	return &SecurityTCPSocket{addr: addr, status: status_Init}
}

func NewSecurityServerTCPSocket(conn net.Conn) *SecurityTCPSocket {
	return &SecurityTCPSocket{conn: conn, status: status_Init}
}

func (s *SecurityTCPSocket) Addr() string {
	return s.addr
}

func (s *SecurityTCPSocket) Connect() error {
	if s.status != status_Init {
		return errors.New("socket has connected")
	}
	s.status = status_Init
	var err error
	s.conn, err = net.Dial("tcp", s.addr)
	if err != nil {
		fmt.Println("SecurityTCPSocket Connect error:", err)
		return err
	}

	fmt.Println("send hello")
	// 1. say hello to server
	sendData := newHelloBuffer()
	_, err = s.writeAll(sendData)
	if err != nil {
		s.conn.Close()
		fmt.Println("SecurityTCPSocket Connect error:", err)
	}

	fmt.Println("send key")
	// 2. read public key from server
	f, _, err := s.readNextFrame()
	fmt.Println("read next ok")
	if err != nil && err != io.EOF {
		s.conn.Close()
		fmt.Println("SecurityTCPSocket recv publicRSAKey error:", err)
		return err
	}
	s.publicRSAKey = *f.body

	fmt.Println("send aes")
	// 3. send aesKey to server
	s.aesKey = s.randomAESKey(16)
	sendData = newAuthkeyBuffer(s.publicRSAKey, s.aesKey)
	_, err = s.writeAll(sendData)
	if err != nil {
		fmt.Println("SecurityTCPSocket send aesKey error:", err)
		return err
	}
	s.status = status_Transfering
	return nil
}

func (s *SecurityTCPSocket) WaitingConnect() error {
	if s.status != status_Init {
		return errors.New("socket has waiting success")
	}
	if s.conn == nil {
		return errors.New("conn is null")
	}
	// 1. hello
	fmt.Println("recv hello")
	nextFrame, _, err := s.readNextFrame()
	fmt.Println("recv hello end")
	if err != nil && err != io.EOF {
		fmt.Println("recv hello error: ", err)
		return err
	}
	if nextFrame.flag != requestType_Hello {
		return errors.New("error frame")
	}
	// TODO: auth key and password
	s.publicRSAKey, s.privateRSAKey, err = util.NewRsaKey(1024)
	if err != nil {
		return err
	}

	// 2. reply key
	fmt.Println("send reply")
	replyData := newReplyBuffer(s.publicRSAKey)
	_, err = s.writeAll(replyData)
	if err != nil {
		return err
	}

	// 3. recv aesKey
	nextFrame, _, err = s.readNextFrame()
	if nextFrame.flag != requestType_AuthKey {
		return errors.New("error frame")
	}

	decryptData, err := util.RsaPrivateKeyDecrypt(&s.privateRSAKey, nextFrame.body)
	if err != nil {
		return err
	}
	s.aesKey = decryptData
	s.status = status_Transfering
	return nil
}

func (s *SecurityTCPSocket) randomAESKey(bits int) []byte {
	str := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	strLength := len(str)
	var aesKey []byte = make([]byte, bits)
	for i := 0; i < bits; i++ {
		aesKey[i] = str[rand.Int()%strLength]
	}
	fmt.Println("aesKey buffer: ", aesKey)
	fmt.Println("aesKey buffer: ", string(aesKey))
	return aesKey
}

func (s *SecurityTCPSocket) readNextFrame() (*frame, bool, error) {
	if s.conn == nil {
		fmt.Println("SecurityTCPSocket Please call Connect first")
		return nil, false, errors.New("empty socket")
	}

	fmt.Println("ccc")
	readFrom, haxNext, err := s.readNextFrameFromCache()
	fmt.Println("ddd")
	if err != nil {
		fmt.Println("SecurityTCPSocket readNextFrameFromCache error:", err)
	}
	if readFrom == nil {
		fmt.Println("readFrom == nil")
		// no whole frame in cache
		var haxNextFrame bool = false
		buf := GetBufferPool().GetBuffer(kCacheBufferSize)
		defer GetBufferPool().PutBuffer(buf)
		for {
			fmt.Println("before read")
			readSize, err := s.conn.Read(*buf)
			fmt.Println("end read")
			fmt.Println("err: ", err)
			if err != nil && err != io.EOF {
				fmt.Println("read error:", err)
				return nil, false, err
			}
			if readSize == 0 && err == io.EOF {
				return nil, false, err
			}
			fmt.Println("readSize: ", readSize)

			var hasNextBuffer bool = true
			if err == io.EOF {
				hasNextBuffer = false
			}
			if s.cacheBuffer.Len() > 10 {
				fmt.Println("bytes:", s.cacheBuffer.Bytes())
			}
			s.cacheBuffer.Write((*buf)[:readSize])
			if s.cacheBuffer.Len() > 10 {
				fmt.Println("bytes:", s.cacheBuffer.Bytes())
			}
			fmt.Println("len: ", s.cacheBuffer.Len())

			var readNextBuffer bool = false
			for {
				fmt.Println("fff")
				nextFrame, hasNextFrameFromCache, err := s.readNextFrameFromCache()
				haxNextFrame = hasNextFrameFromCache
				if err != nil {
					fmt.Println("SecurityTCPSocket read next frame error:", err)
					return nil, hasNextFrameFromCache, err
				}
				if nextFrame == nil {
					fmt.Println("nextFrom is nil")
					readNextBuffer = true
					break
				}
				fmt.Println("eee")
				if !hasNextBuffer && !hasNextFrameFromCache {
					fmt.Println("current is eof")
					return nextFrame, hasNextFrameFromCache, io.EOF
				}
				fmt.Println("xxy")
				return nextFrame, hasNextFrameFromCache, nil
			}
			if !hasNextBuffer || !readNextBuffer {
				break
			}
		}
		return nil, haxNextFrame, nil
	}
	fmt.Println("xxxx")
	return readFrom, haxNext, nil
}

func (s *SecurityTCPSocket) readNextFrameFromCache() (*frame, bool, error) {
	if s.cacheBuffer.Len() == 0 {
		return nil, false, nil
	}
	var frameLength int = 0
	if s.cacheBuffer.Len() >= 4 {
		frameLength = (int(s.cacheBuffer.Bytes()[0]) << 24) | (int(s.cacheBuffer.Bytes()[1]) << 16) |
			(int(s.cacheBuffer.Bytes()[2]) << 8) | int(s.cacheBuffer.Bytes()[3])
		fmt.Println("cache frameLength: ", frameLength)
		fmt.Println("cache flag: ", s.cacheBuffer.Bytes()[1])
		fmt.Println("s.cacheBuffer.Bytes(): ", s.cacheBuffer.Bytes()[:4])
	} else {
		return nil, false, nil
	}
	// frameLength, err := s.cacheBuffer.ReadByte()
	// if err != nil {
	// 	fmt.Println("SecurityTCPSocket cache read length error:", err)
	// 	return nil, false, err
	// }
	if s.cacheBuffer.Len() < int(frameLength) {
		return nil, true, nil
	}
	var cacheLength []byte = make([]byte, 4)
	s.cacheBuffer.Read(cacheLength)
	frameBuffer := GetBufferPool().GetBuffer(int(frameLength))
	defer GetBufferPool().PutBuffer(frameBuffer)
	s.cacheBuffer.Read(*frameBuffer)

	if s.cacheBuffer.Len() > 10 {
		fmt.Println("remain: ", s.cacheBuffer.Bytes()[:10])
	}
	fmt.Println("frameLength: ", frameLength)
	fmt.Println("aesKey: ", s.aesKey)
	nextFrame, err := unmarshalFrame(int(frameLength), frameBuffer, s.aesKey)
	var hasNextFrame bool = s.cacheBuffer.Len() > 0
	return nextFrame, hasNextFrame, err
}

func (s *SecurityTCPSocket) Read(readData []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	fmt.Println("call read")
	if s.conn == nil {
		fmt.Println("SecurityTCPSocket Please call Connect first")
		return 0, errors.New("empty socket")
	}
	readDataSize := len(readData)
	if s.dataBuffer.Len() >= readDataSize {
		s.dataBuffer.Read(readData)
		return readDataSize, nil
	}
	fmt.Println("bbb")
	for {
		fmt.Println("before readNextFrame")
		nextFrame, hasNextCacheFrom, err := s.readNextFrame()
		fmt.Println("aaa")
		if err != nil && err != io.EOF {
			fmt.Println("SecurityTCPSocket Read error:", err)
			return 0, err
		}
		if nextFrame == nil {
			fmt.Println("err: ", err)
		}
		if nextFrame == nil && err == io.EOF {
			fmt.Println("read complete")
			if s.dataBuffer.Len() > 0 {
				readSize, err := s.dataBuffer.Read(readData)
				return readSize, err
			}
			return 0, nil
		}
		// if err == io.EOF {
		// 	eofReadSize, _ := s.dataBuffer.Read(readData)
		// 	return eofReadSize, io.EOF
		// }
		fmt.Println("nextFrame: ", nextFrame)
		switch nextFrame.flag {
		case requestType_Ping:
			continue
		case reqeustType_NewRequest:
			if s.frameIndex == 0 && s.dataBuffer.Len() == 0 {
				fmt.Println("new request")
			} else {
				panic("error frame")
			}
		case reqeustType_Content:
			fmt.Println("content")
			s.frameIndex = s.frameIndex + 1
			s.dataBuffer.Write(*nextFrame.body)
			if s.dataBuffer.Len() >= readDataSize || err == io.EOF || !hasNextCacheFrom {
				bufferReadSize, _ := s.dataBuffer.Read(readData)
				fmt.Println("read over")
				return bufferReadSize, err
			}
		case requestType_Close:
			fmt.Println("requestType_Close")
			s.status = status_Closed
			s.frameIndex = 0
			currentDataBufferSize := s.dataBuffer.Len()
			s.dataBuffer.Read(readData)
			return currentDataBufferSize, Close
		default:
			panic("error frame")
		}
	}
}

func (s *SecurityTCPSocket) Write(writeData []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	writeDataSize := len(writeData)
	totalDataSize := writeDataSize
	var writeBuffer bytes.Buffer
	fmt.Println("before write")
	writeBuffer.Write(writeData)
	fmt.Println("111")

	var cacheBuffer = GetBufferPool().GetBuffer(kCacheBufferSize)
	for writeDataSize > 0 {
		fmt.Println("333")
		readSize, _ := writeBuffer.Read(*cacheBuffer)
		contentBuffer := (*cacheBuffer)[:readSize]
		frameBuffer := newContentBuffer(&contentBuffer, string(s.aesKey))
		writedSize, err := s.writeAll(frameBuffer)
		if err != nil {
			return totalDataSize - writedSize, err
		}
		writeDataSize = writeDataSize - readSize
		fmt.Println("writeDataSize: ", writeDataSize)
	}
	fmt.Println("222")
	return 0, nil
}

func (s *SecurityTCPSocket) Close() {
	s.status = status_Closed
	fmt.Println("SecurityTCPSocket Close")
	// only send close pacakge, do not close tcp channel.
	buffer := newCloseBuffer(string(s.aesKey))
	s.writeAll(buffer)
}

func (s *SecurityTCPSocket) Refresh() {
	if s.status == status_Closed {
		fmt.Println("SecurityTCPSocket Refresh")
		buffer := newNewRequestBuffer(string(s.aesKey))
		s.writeAll(buffer)
	} else {
		fmt.Println("now socekt is not closed")
	}
}

func (s *SecurityTCPSocket) Stop() {
	s.status = status_Closed
	s.conn.Close()
	s.conn = nil
}

func (s *SecurityTCPSocket) writeAll(data []byte) (int, error) {
	var totalWriteSize int = 0
	var dataSize int = len(data)
	for {
		writeSize, err := s.conn.Write(data)
		if err != nil && err != io.EOF {
			fmt.Println("SecurityTCPSocket write failed:", err)
			return totalWriteSize, err
		}
		totalWriteSize += writeSize
		if totalWriteSize == dataSize {
			break
		}
		if err == io.EOF {
			break
		}
	}
	return totalWriteSize, nil
}

func (s *SecurityTCPSocket) Ping() {
	buffer := newPingBuffer(string(s.aesKey))
	s.writeAll(buffer)
}
