package socket

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
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
	status_Stoped      = iota
)

const (
	signalType_Timer      = iota
	signalType_WriteEvent = iota
)

func SecurityTCPSocketListen(addr string) (*SecurityTCPSocket, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	sock := newSecurityServerTCPSocket(listener)
	return sock, err
}

type SecurityTCPSocket struct {
	// about connect
	addr                string
	conn                net.Conn
	isTCPReady          bool
	acceptTCPSocketList *list.List
	listenTCPSocket     *SecurityTCPSocket
	status              int
	stopError           error

	// tcp cache buffer
	cacheBuffer bytes.Buffer

	// tcp real read data buffer
	readDataBuffer  bytes.Buffer
	writeDataBuffer bytes.Buffer

	// about security
	publicRSAKey  []byte
	privateRSAKey []byte
	aesKey        []byte
	frameIndex    int

	securityListener SecurityListener
	socketListener   net.Listener
	// locker
	locker      *sync.Mutex
	cond        *sync.Cond
	writeLocker *sync.Mutex
	signalType  int
	// timer
	sendTimer *util.Timer
}

func NewSecurityTCPSocket(addr string) *SecurityTCPSocket {
	return &SecurityTCPSocket{addr: addr, status: status_Init}
}

func newSecurityServerTCPSocket(sid int, listeneSocket Listener) *SecurityTCPSocket {
	return nil
}

func newSecurityServerTCPSocket(listener net.Listener) *SecurityTCPSocket {
	return &SecurityTCPSocket{socketListener: listener, status: status_Init}
}

func newSecurityServerTCPSocketFromNetConn(conn net.Conn) *SecurityTCPSocket {
	return &SecurityTCPSocket{conn: conn, status: status_Init}
}

func (s *SecurityTCPSocket) SetListener(listener SecurityListener) {
	s.securityListener = listener
}

func (s *SecurityTCPSocket) Status() int {
	return s.status
}

func (s *SecurityTCPSocket) Addr() string {
	return s.addr
}

func (s *SecurityTCPSocket) Accept() (Socket, error) {
	if s.acceptTCPSocketList != nil {
		// 1. get from idle list
		for iter := s.acceptTCPSocketList.Front(); iter != nil; iter = iter.Next() {
			acceptTCPSocket := iter.Value.(*SecurityTCPSocket)
			if acceptTCPSocket.Status() == status_Closed {
				s.acceptTCPSocketList.Remove(iter)
				return acceptTCPSocket, nil
			}
		}
	}
	// 2. get from conn.Accept
	acceptConn, err := s.socketListener.Accept()
	if err != nil {
		fmt.Println("SecurityTCPSocket accept error:", err)
		return nil, err
	}
	securityTCPSocket := newSecurityServerTCPSocketFromNetConn(acceptConn)
	if s.acceptTCPSocketList == nil {
		s.acceptTCPSocketList = list.New()
	}
	s.acceptTCPSocketList.PushBack(securityTCPSocket)
	securityTCPSocket.listenTCPSocket = s
	securityTCPSocket.waitingConnect()
	securityTCPSocket.startBackgroundReadTask()
	securityTCPSocket.startBackgroundWriteTask()
	return securityTCPSocket, nil
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

	// 1. say hello to server
	sendData := newHelloBuffer()
	_, err = s.writeAll(sendData)
	if err != nil {
		s.conn.Close()
		fmt.Println("SecurityTCPSocket Connect error:", err)
	}

	// 2. read public key from server
	f, err := s.readNextFrame()
	if err != nil && err != io.EOF {
		s.conn.Close()
		fmt.Println("SecurityTCPSocket recv publicRSAKey error:", err)
		return err
	}
	s.publicRSAKey = *f.body

	// 3. send aesKey to server
	s.aesKey = s.randomAESKey(16)
	sendData = newAuthkeyBuffer(s.publicRSAKey, s.aesKey)
	_, err = s.writeAll(sendData)
	if err != nil {
		fmt.Println("SecurityTCPSocket send aesKey error:", err)
		return err
	}
	s.status = status_Transfering
	s.startBackgroundReadTask()
	s.startBackgroundWriteTask()
	fmt.Println("connect success")
	return nil
}

func (s *SecurityTCPSocket) waitingConnect() error {
	if s.status != status_Init {
		return errors.New("socket has waiting success")
	}
	if s.conn == nil {
		return errors.New("conn is null")
	}
	// 1. hello
	nextFrame, err := s.readNextFrame()
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
	replyData := newReplyBuffer(s.publicRSAKey)
	_, err = s.writeAll(replyData)
	if err != nil {
		return err
	}

	// 3. recv aesKey
	nextFrame, err = s.readNextFrame()
	if nextFrame.flag != requestType_AuthKey {
		return errors.New("error frame")
	}

	decryptData, err := util.RsaPrivateKeyDecrypt(&s.privateRSAKey, nextFrame.body)
	if err != nil {
		return err
	}
	s.aesKey = decryptData
	s.status = status_Transfering
	fmt.Println("waiting connect success")
	return nil
}

func (s *SecurityTCPSocket) randomAESKey(bits int) []byte {
	str := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	strLength := len(str)
	var aesKey []byte = make([]byte, bits)
	for i := 0; i < bits; i++ {
		aesKey[i] = str[rand.Int()%strLength]
	}
	return aesKey
}

func (s *SecurityTCPSocket) readNextFrame() (*frame, error) {
	if s.conn == nil {
		fmt.Println("SecurityTCPSocket Please call Connect first")
		return nil, errors.New("empty socket")
	}

	nextFrame, err := s.readNextFrameFromCache()
	if nextFrame != nil {
		return nextFrame, nil
	}
	if err != nil {
		return nil, err
	}

	var readBuffer *[]byte = GetBufferPool().GetBuffer(kCacheBufferSize)
	defer GetBufferPool().PutBuffer(readBuffer)
	for {
		readCount, err := s.conn.Read(*readBuffer)
		if err != nil && err != io.EOF {
			// error with socket
			return nil, err
		}
		if readCount != 0 {
			s.cacheBuffer.Write((*readBuffer)[:readCount])
		}
		nextFrame, err = s.readNextFrameFromCache()
		if err != nil {
			fmt.Println("SecurityTCPSocket startBackgroundReadTask nextFrame error:", err)
			s.stopError = err
			return nil, err
		}
		if nextFrame != nil {
			return nextFrame, nil
		}
	}
}

func (s *SecurityTCPSocket) readNextFrameFromCache() (*frame, error) {
	if s.cacheBuffer.Len() == 0 {
		return nil, nil
	}
	var frameLength int = 0
	if s.cacheBuffer.Len() >= 4 {
		frameLength = (int(s.cacheBuffer.Bytes()[0]) << 24) | (int(s.cacheBuffer.Bytes()[1]) << 16) |
			(int(s.cacheBuffer.Bytes()[2]) << 8) | int(s.cacheBuffer.Bytes()[3])
	} else {
		return nil, nil
	}
	if s.cacheBuffer.Len() < int(frameLength) {
		return nil, nil
	}
	var cacheLength []byte = make([]byte, 4)
	s.cacheBuffer.Read(cacheLength)
	frameBuffer := GetBufferPool().GetBuffer(int(frameLength))
	defer GetBufferPool().PutBuffer(frameBuffer)
	s.cacheBuffer.Read(*frameBuffer)

	nextFrame, err := unmarshalFrame(frameLength, frameBuffer, s.aesKey)
	return nextFrame, err
}

func (s *SecurityTCPSocket) Read(readData []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	if s.conn == nil {
		fmt.Println("SecurityTCPSocket Please call Connect first")
		return 0, errors.New("empty socket")
	}
	if s.stopError != nil {
		return 0, s.stopError
	}
	currentReadDataLength := len(readData)
	if s.readDataBuffer.Len() < currentReadDataLength {
		currentReadDataLength = s.readDataBuffer.Len()
	}
	readSize, err := s.readDataBuffer.Read(readData)
	return readSize, err
}

func (s *SecurityTCPSocket) Write(writeData []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	s.writeLocker.Lock()
	s.writeDataBuffer.Write(writeData)
	s.writeLocker.Unlock()
	s.signalType = signalType_WriteEvent
	s.cond.Signal()

	return len(writeData), nil
}

func (s *SecurityTCPSocket) Close() {
	s.status = status_Closed
	fmt.Println("SecurityTCPSocket Close")
	// only send close pacakge, do not close tcp channel.
	buffer := newCloseBuffer(string(s.aesKey))
	s.writeAll(buffer)
	if s.securityListener != nil {
		s.securityListener.OnClose(s)
	}
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

func (s *SecurityTCPSocket) Ping() {
	buffer := newPingBuffer(string(s.aesKey))
	s.writeAll(buffer)
}

func (s *SecurityTCPSocket) startBackgroundWriteTask() {
	s.locker = new(sync.Mutex)
	s.cond = sync.NewCond(s.locker)
	s.writeLocker = new(sync.Mutex)
	s.sendTimer = util.NewOneShotTimer()
	go func() {
		for {
			// waiting for singal
			s.cond.L.Lock()
			s.cond.Wait()
			s.sendTimer.Stop()
			var writeBuffer *[]byte = nil
			var nextFrameSize int = kFrameBufferSize
			var nextMinFrameSize int = kFrameBufferSize
			for {
				s.writeLocker.Lock()
				if s.writeDataBuffer.Len() >= nextMinFrameSize {
					if writeBuffer == nil {
						writeBuffer = GetBufferPool().GetBuffer(nextFrameSize)
					}
					s.writeDataBuffer.Read(*writeBuffer)
					s.writeInBackground(*writeBuffer)
					if nextMinFrameSize == 0 {
						s.writeLocker.Unlock()
						break
					}
				} else {
					if s.signalType == signalType_Timer {
						GetBufferPool().PutBuffer(writeBuffer)
						writeBuffer = nil
						nextMinFrameSize = 0
						nextFrameSize = s.writeDataBuffer.Len()
						s.writeLocker.Unlock()
						continue
					}
					// sleep 2ms, waiting for buffer
					if s.writeDataBuffer.Len() == 0 {
						s.writeLocker.Unlock()
						break
					}
					s.writeLocker.Unlock()
					s.sendTimer.Start(time.Second*1, func() {
						s.signalType = signalType_Timer
						s.cond.Signal()
					})
					break
				}
				s.writeLocker.Unlock()
			}
			s.cond.L.Unlock()
			GetBufferPool().PutBuffer(writeBuffer)
		}
	}()
}

func (s *SecurityTCPSocket) startBackgroundReadTask() {
	go func() {
		for {
			nextFrame, err := s.readNextFrame()
			if err != nil {
				fmt.Println("SecurityTCPSocket startBackgroundReadTask nextFrame error:", err)
				s.stopError = err
				return
			}
			if nextFrame == nil {
				break
			}
			switch nextFrame.flag {
			case requestType_Ping:
				fmt.Println("ping")
				if s.status != status_Closed {
					s.stopError = errors.New("socket is not closed")
					return
				}
			case reqeustType_Content:
				if s.status != status_Transfering {
					s.stopError = errors.New("socket is not working")
					return
				}
				s.readDataBuffer.Write(*nextFrame.body)
			case requestType_Close:
				fmt.Println("socket closed")
				s.stopError = Close
			case requestType_Error:
				s.stopError = errors.New(string(*nextFrame.body))
			case reqeustType_NewRequest:
				if s.status != status_Closed {
					s.stopError = errors.New("socket is not closed")
					return
				}
				s.stopError = nil
				s.readDataBuffer.Reset()
				s.writeDataBuffer.Reset()
				s.cacheBuffer.Reset()
			}
		}
	}()
}

func (s *SecurityTCPSocket) writeInBackground(writeData []byte) {
	writeDataSize := len(writeData)
	var writeBuffer bytes.Buffer
	writeBuffer.Write(writeData)

	var cacheBuffer = GetBufferPool().GetBuffer(kCacheBufferSize)
	for writeDataSize > 0 {
		readSize, _ := writeBuffer.Read(*cacheBuffer)
		contentBuffer := (*cacheBuffer)[:readSize]
		frameBuffer := newContentBuffer(&contentBuffer, string(s.aesKey))
		_, err := s.writeAll(frameBuffer)
		if err != nil {
			fmt.Println("writeAll error:", err)
			return
		}
		writeDataSize = writeDataSize - readSize
	}
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
