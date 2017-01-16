package socket

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
	"util"
)

func NewRealSecurityTCPSocket(tcpConn net.Conn) *RealSecurityTCPSocket {
	return &RealSecurityTCPSocket{tcpConn: tcpConn}
}

func NewRealSecurityTCPSocketWithAddr(addr string) *RealSecurityTCPSocket {
	return &RealSecurityTCPSocket{addr: addr}
}

type RealSecurityTCPSocket struct {
	// origin net data
	addr    string
	tcpConn net.Conn

	currentSid            int
	allAcceptSocket       map[int]*VirtualSecurityTCPSocket
	allAcceptSocketLocker *sync.RWMutex
	cacheBuffer           bytes.Buffer
	acceptLocker          *sync.Mutex
	acceptCond            *sync.Cond

	// about security
	publicRSAKey  []byte
	privateRSAKey []byte
	aesKey        []byte
	frameIndex    int

	// locker
	locker *sync.Mutex
	cond   *sync.Cond

	status     int
	signalType int
}

func (s *RealSecurityTCPSocket) Close() {
	s.tcpConn.Close()
}

func (s *RealSecurityTCPSocket) Connect() error {
	if s.status != status_Init {
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
	return nil
}

func (r *RealSecurityTCPSocket) NewVirtualSocket() Socket {
	if r.allAcceptSocket == nil {
		r.allAcceptSocket = make(map[int]*VirtualSecurityTCPSocket)
		r.allAcceptSocketLocker = new(sync.RWMutex)
	}
	virtualSid := r.currentSid
	r.currentSid = r.currentSid + 1
	virtualSocket := NewVirtualSecurityTCPSocket(virtualSid, r)
	r.allAcceptSocketLocker.Lock()
	r.allAcceptSocket[virtualSid] = virtualSocket
	r.allAcceptSocketLocker.Unlock()
	return virtualSocket
}

func (r *RealSecurityTCPSocket) NewClientVirtualSocket() Socket {
	if r.currentSid == 0 {
		return r.NewVirtualSocket()
	}

	virtualSocket := r.NewVirtualSocket()
	r.NewRequest()
	return virtualSocket
}

func (s *RealSecurityTCPSocket) NewRequest() {
	if s.status == status_Transfering {
		fmt.Println("VirtualSecurityTCPSocket NewRequest")
		buffer := newNewRequestBuffer(string(s.aesKey))
		s.writeAll(buffer)
	} else {
		fmt.Println("now socekt is not working")
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
			fmt.Println("startBackgroundTask")
			var needNotify bool = false
			var notifySidMap map[int]bool = make(map[int]bool)
			nextFrame, err := s.readNextFrame()
			for {
				if err != nil {
					fmt.Println("SecurityTCPSocket startBackgroundReadTask nextFrame error:", err)
					return
				}
				if nextFrame == nil {
					break
				}
				switch nextFrame.flag {
				case requestType_Ping:
					fmt.Println("ping")
					if s.status != status_Closed {
						return
					}
				default:
					stsFrame := translateToStsFrame(nextFrame)
					s.allAcceptSocketLocker.Lock()
					frameSocket, ok := s.allAcceptSocket[stsFrame.sid]
					if !ok {
						fmt.Println("not exist socket")
						s.allAcceptSocketLocker.Unlock()
						break
					}
					switch stsFrame.flag {
					case reqeustType_Content:
						if s.status != status_Transfering {
							fmt.Println("status error")
							frameSocket.stopError = errors.New("socket is not working")
							s.allAcceptSocketLocker.Unlock()
							break
						}
						frameSocket.readLocker.Lock()
						frameSocket.readDataBuffer.Write((*nextFrame.body)[nextFrame.pos:])
						frameSocket.readLocker.Unlock()
						needNotify = true
						notifySidMap[stsFrame.sid] = true
					case requestType_Close:
						fmt.Println("socket closed")
						frameSocket.stopError = Close
						delete(s.allAcceptSocket, stsFrame.sid)
					case requestType_Error:
						frameSocket.stopError = errors.New(string(*nextFrame.body))
						delete(s.allAcceptSocket, stsFrame.sid)
					case reqeustType_NewRequest:
						if s.status != status_Transfering {
							fmt.Println("status error:", s.status)
							frameSocket.stopError = errors.New("socket is not closed")
							s.allAcceptSocketLocker.Unlock()
							break
						}
						fmt.Println("singal")
						frameSocket.stopError = nil
						frameSocket.readDataBuffer.Reset()
						frameSocket.writeDataBuffer.Reset()
						frameSocket.cacheBuffer.Reset()

						s.acceptCond.Signal()
					}
					s.allAcceptSocketLocker.Unlock()
				}
				nextFrame, err = s.readNextFrameFromCache()
			}
			if needNotify {
				s.allAcceptSocketLocker.RLock()
				for sid, _ := range notifySidMap {
					frameSocket := s.allAcceptSocket[sid]
					frameSocket.readNotifyCond.Signal()
				}
				s.allAcceptSocketLocker.RUnlock()
			}
		}
	}()
}

func (s *RealSecurityTCPSocket) startOneWriteTask(sid int) {
	s.allAcceptSocketLocker.RLock()
	sock := s.allAcceptSocket[sid]
	s.allAcceptSocketLocker.RUnlock()
	sock.sendTimer.Stop()
	var writeBuffer *[]byte = nil
	var nextFrameSize int = kFrameBufferSize
	var nextMinFrameSize int = kFrameBufferSize
	sock.writeLocker.Lock()
	for {
		if sock.hasWriteData() {
			if sock.writeDataBuffer.Len() >= nextMinFrameSize {
				if writeBuffer == nil {
					writeBuffer = GetBufferPool().GetBuffer(nextFrameSize)
				}
				sock.writeDataBuffer.Read(*writeBuffer)
				s.writeInBackground(sid, *writeBuffer)
				if nextMinFrameSize == 0 {
					break
				}
			} else {
				if s.signalType == signalType_Timer {
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
					s.signalType = signalType_Timer
					s.cond.Signal()
				})
				break
			}
		} else {
			break
		}
	}
	sock.writeLocker.Unlock()
	GetBufferPool().PutBuffer(writeBuffer)
}

func (s *RealSecurityTCPSocket) startBackgroundWriteTask() {
	s.locker = new(sync.Mutex)
	s.cond = sync.NewCond(s.locker)
	go func() {
		for {
			fmt.Println("startbackgroundWriteTask")
			// waiting for singal
			s.cond.L.Lock()
			s.cond.Wait()
			s.allAcceptSocketLocker.RLock()
			for sid, _ := range s.allAcceptSocket {
				s.startOneWriteTask(sid)
			}
			s.allAcceptSocketLocker.RUnlock()
			s.cond.L.Unlock()
		}
	}()
}

func (s *RealSecurityTCPSocket) writeInBackground(sid int, writeData []byte) {
	writeDataSize := len(writeData)
	var writeBuffer bytes.Buffer
	writeBuffer.Write(writeData)

	var cacheBuffer = GetBufferPool().GetBuffer(kCacheBufferSize)
	for writeDataSize > 0 {
		readSize, _ := writeBuffer.Read(*cacheBuffer)
		contentBuffer := (*cacheBuffer)[:readSize]
		frameBuffer := newContentBuffer(sid, &contentBuffer, string(s.aesKey))
		_, err := s.writeAll(frameBuffer)
		if err != nil {
			fmt.Println("writeAll error:", err)
			return
		}
		writeDataSize = writeDataSize - readSize
	}
}

func (s *RealSecurityTCPSocket) writeVirtualData(sid int, data []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	s.allAcceptSocketLocker.RLock()
	if _, ok := s.allAcceptSocket[sid]; !ok {
		return 0, errors.New("invalid sid")
	}
	virtualSocket := s.allAcceptSocket[sid]
	s.allAcceptSocketLocker.RUnlock()
	if virtualSocket.stopError != nil {
		return 0, virtualSocket.stopError
	}
	virtualSocket.writeLocker.Lock()
	virtualSocket.writeDataBuffer.Write(data)
	virtualSocket.writeLocker.Unlock()
	s.signalType = signalType_WriteEvent
	s.cond.Signal()

	return len(data), nil
}

func (s *RealSecurityTCPSocket) readVirtualData(sid int, data []byte) (int, error) {
	if s.status != status_Transfering {
		return 0, errors.New("socket is not working")
	}
	s.allAcceptSocketLocker.RLock()
	if _, ok := s.allAcceptSocket[sid]; !ok {
		return 0, errors.New("invalid sid")
	}
	virtualSocket := s.allAcceptSocket[sid]
	s.allAcceptSocketLocker.RUnlock()
	if virtualSocket.stopError != nil {
		return 0, virtualSocket.stopError
	}

	virtualSocket.readNotifyCond.L.Lock()
	virtualSocket.readNotifyCond.Wait()
	defer virtualSocket.readNotifyCond.L.Unlock()
	virtualSocket.readLocker.Lock()
	defer virtualSocket.readLocker.Unlock()
	if virtualSocket.stopError != nil {
		return 0, virtualSocket.stopError
	}
	currentReadDataLength := len(data)
	if virtualSocket.readDataBuffer.Len() < currentReadDataLength {
		currentReadDataLength = virtualSocket.readDataBuffer.Len()
	}
	readSize, err := virtualSocket.readDataBuffer.Read(data)
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

func (r *RealSecurityTCPSocket) closeSocketBySid(sid int) {
	r.allAcceptSocketLocker.Lock()
	if _, ok := r.allAcceptSocket[sid]; ok {
		buffer := newCloseBuffer(sid, string(r.aesKey))
		r.writeAll(buffer)
		delete(r.allAcceptSocket, sid)
	}
	r.allAcceptSocketLocker.Unlock()
}
