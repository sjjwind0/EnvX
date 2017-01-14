package socket

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

func SecurityTCPSocketListen(addr string) (*VirtualSecurityTCPSocket, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	sock := NewRealSecurityTCPSocket(addr, listener)
	return sock, err
}

func NewSecurityTCPSocket() (*RealSecurityTCPSocket, error) {
	return &RealSecurityTCPSocket
}

func NewRealSecurityTCPSocket(addr string, netListener net.Listener) {
	return &RealSecurityTCPSocket{addr: addr, netListener: netListener}
}

type RealSecurityTCPSocket struct {
	netListener net.Listener
	tcpConn     net.Conn

	addr            string
	currentSid      int
	allAcceptSocket map[int]*VirtualSecurityTCPSocket

	// about security
	publicRSAKey  []byte
	privateRSAKey []byte
	aesKey        []byte
	frameIndex    int

	// locker
	locker *sync.Mutex
	cond   *sync.Cond
}

func (s *RealSecurityTCPSocket) Accept() (Socket, error) {
	if s.allAcceptSocket == nil {
		s.allAcceptSocket = make(map[int]*VirtualSecurityTCPSocket)

		var err error = nil
		s.tcpConn, err = s.netListener.Accept()
		if err != nil {
			fmt.Println("RealSecurityTCPSocket Accept failed:", err)
			return nil, err
		}
		err = s.waitingConnect()
		if err != nil {
			return nil, err
		}
	}
	return s.NewVirtualSocket(), nil
}

func (s *RealSecurityTCPSocket) Addr() string {
	return s.addr
}

func (s *RealSecurityTCPSocket) Close() {
	s.tcpConn.Close()
}

func (s *RealSecurityTCPSocket) Connect() error {
	if s.status == status_Init {
		return errors.New("socket has connected")
	}
	s.status = status_Init
	var err error
	s.tcpConn, err = net.Dial("tcp", s.addr)
	if err != nil {
		fmt.Println("SecurityTCPSocket Connect error:", err)
		return err
	}

	// 1. say hello to server
	sendData := newHelloBuffer()
	_, err = s.writeAll(sendData)
	if err != nil {
		s.tcpConn.Close()
		fmt.Println("SecurityTCPSocket Connect error:", err)
	}

	// 2. read public key from server
	f, err := s.readNextFrame()
	if err != nil && err != io.EOF {
		s.tcpConn.Close()
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

func (r *RealSecurityTCPSocket) NewVirtualSocket() Socket {
	virtualSid := currentSid
	s.currentSid = s.currentSid + 1
	virtualSocket := newSecurityServerTCPSocket(virtualSid, s)
	s.allAcceptSocket[virtualSid] = virtualSocket
	return virtualSocket
}

func (s *RealSecurityTCPSocket) NewRequest() {
	if s.status == status_Closed {
		fmt.Println("VirtualSecurityTCPSocket NewRequest")
		buffer := newNewRequestBuffer(string(s.aesKey))
		s.writeAll(buffer)
	} else {
		fmt.Println("now socekt is not closed")
	}
}

func (s *RealSecurityTCPSocket) Stop() {
	s.status = status_Closed
	s.tcpConn.Close()
	s.tcpConn = nil
}

func (s *RealSecurityTCPSocket) Ping() {
	buffer := newPingBuffer(string(s.aesKey))
	s.writeAll(buffer)
}

func (s *RealSecurityTCPSocket) waitingConnect() error {
	if s.status != status_Init {
		return errors.New("socket has waiting success")
	}
	if s.tcpConn == nil {
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

func (s *RealSecurityTCPSocket) randomAESKey(bits int) []byte {
	str := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	strLength := len(str)
	var aesKey []byte = make([]byte, bits)
	for i := 0; i < bits; i++ {
		aesKey[i] = str[rand.Int()%strLength]
	}
	return aesKey
}

func (s *RealSecurityTCPSocket) readNextFrame() (*frame, error) {
	if s.tcpConn == nil {
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
		readCount, err := s.tcpConn.Read(*readBuffer)
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

func (s *RealSecurityTCPSocket) readNextFrameFromCache() (*frame, error) {
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

func (s *RealSecurityTCPSocket) startBackgroundReadTask() {
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
			default:
				stsFrame := translateToStsFrame(nextFrame)
				frameSocket := s.allAcceptSocket[stsFrame.sid]
				switch stsFrame.flag {
				case reqeustType_Content:
					if frameSocket.status != status_Transfering {
						frameSocket.stopError = errors.New("socket is not working")
						return
					}
					frameSocket.readDataBuffer.Write(*nextFrame.body)
				case requestType_Close:
					fmt.Println("socket closed")
					frameSocket.stopError = Close
				case requestType_Error:
					frameSocket.stopError = errors.New(string(*nextFrame.body))
				case reqeustType_NewRequest:
					if frameSocket.status != status_Closed {
						frameSocket.stopError = errors.New("socket is not closed")
						return
					}
					frameSocket.stopError = nil
					frameSocket.readDataBuffer.Reset()
					frameSocket.writeDataBuffer.Reset()
					frameSocket.cacheBuffer.Reset()
				}
			}
		}
	}()
}

func (s *RealSecurityTCPSocket) startOneWriteTask(sid int) {
	sock := s.allAcceptSocket[sid]
	sock.sendTimer.Stop()
	var writeBuffer *[]byte = nil
	var nextFrameSize int = kFrameBufferSize
	var nextMinFrameSize int = kFrameBufferSize
	for {
		sock.writeLocker.Lock()
		if sock.hasWriteData() {
			if sock.writeDataBuffer.Len() >= nextMinFrameSize {
				if writeBuffer == nil {
					writeBuffer = GetBufferPool().GetBuffer(nextFrameSize)
				}
				sock.writeDataBuffer.Read(*writeBuffer)
				sock.writeInBackground(*writeBuffer)
				if nextMinFrameSize == 0 {
					break
				}
			} else {
				if sock.signalType == signalType_Timer {
					GetBufferPool().PutBuffer(writeBuffer)
					writeBuffer = nil
					nextMinFrameSize = 0
					nextFrameSize = sock.writeDataBuffer.Len()
					continue
				}
				// sleep 2ms, waiting for buffer
				if sock.writeDataBuffer.Len() == 0 {
					break
				}
				sock.sendTimer.Start(time.Second*1, func() {
					sock.signalType = signalType_Timer
					sock.cond.Signal()
				})
				break
			}
		}
		sock.writeLocker.Unlock()
		GetBufferPool().PutBuffer(writeBuffer)
	}
}

func (s *RealSecurityTCPSocket) startBackgroundWriteTask() {
	s.locker = new(sync.Mutex)
	s.cond = sync.NewCond(s.locker)
	s.writeLocker = new(sync.Mutex)
	s.sendTimer = util.NewOneShotTimer()
	go func() {
		for {
			// waiting for singal
			s.cond.L.Lock()
			s.cond.Wait()
			for sid, _ := range s.allAcceptSocket {
				s.startOneWriteTask(sid)
			}
			s.cond.L.Unlock()
		}
	}()
}

func (s *RealSecurityTCPSocket) writeInBackground(writeData []byte) {
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

func (s *RealSecurityTCPSocket) writeVirtualData(sid int, data []byte) (int, error) {
	if _, ok := s.allAcceptSocket[sid]; ok {
		return 0, errors.New("invalid sid")
	}
	virtualSocket := s.allAcceptSocket[sid]
	if virtualSocket.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	virtualSocket.writeLocker.Lock()
	virtualSocket.writeDataBuffer.Write(writeData)
	virtualSocket.writeLocker.Unlock()
	virtualSocket.signalType = signalType_WriteEvent
	s.cond.Signal()

	return len(writeData), nil
}

func (s *RealSecurityTCPSocket) readVirtualData(sid int, data []byte) (int, error) {
	if _, ok := s.allAcceptSocket[sid]; ok {
		return 0, errors.New("invalid sid")
	}
	virtualSocket := s.allAcceptSocket[sid]
	virtualSocket.readLocker.Lock()
	defer virtualSocket.readLocker.Unlock()

	if virtualSocket.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	if virtualSocket.stopError != nil {
		return 0, virtualSocket.stopError
	}
	currentReadDataLength := len(readData)
	if virtualSocket.readDataBuffer.Len() < currentReadDataLength {
		currentReadDataLength = virtualSocket.readDataBuffer.Len()
	}
	readSize, err := virtualSocket.readDataBuffer.Read(readData)
	return readSize, err
}

func (s *RealSecurityTCPSocket) writeAll(data []byte) (int, error) {
	var totalWriteSize int = 0
	var dataSize int = len(data)
	for {
		writeSize, err := s.tcpConn.Write(data)
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
