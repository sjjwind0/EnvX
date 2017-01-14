package socket

import (
	"bytes"
	"fmt"
	"sync"
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

type VirtualSecurityTCPSocket struct {
	// about connect
	sid           int
	realTCPSocket *RealSecurityTCPSocket
	status        int
	stopError     error

	cacheBuffer     bytes.Buffer
	readDataBuffer  bytes.Buffer
	writeDataBuffer bytes.Buffer

	readLocker  sync.Mutex
	writeLocker sync.Mutex

	sendTimer util.Timer

	securityListener SecurityListener
}

func NewVirtualSecurityTCPSocket(sid int, realTCPSocket *securityBasicSocket) *VirtualSecurityTCPSocket {
	return &VirtualSecurityTCPSocket{sid: sid, realTCPSocket: realTCPSocket}
}

func (v *VirtualSecurityTCPSocket) SetListener(listener SecurityListener) {
	v.securityListener = listener
}

func (v *VirtualSecurityTCPSocket) Read(readData []byte) (int, error) {
	readSize, err := realTCPSocket.readVirtualData(v.sid, readData)
	return readSize, err
}

func (v *VirtualSecurityTCPSocket) Write(writeData []byte) (int, error) {
	writeSize, err := v.realTCPSocket.writeVirtualData(v.sid, writeData)
	return writeSize, err
}

func (v *VirtualSecurityTCPSocket) Close() {
	v.status = status_Closed
	fmt.Println("VirtualSecurityTCPSocket Close")
	// only send close pacakge, do not close tcp channel.
	buffer := newCloseBuffer(string(v.aesKey))
	v.writeAll(buffer)
	if v.securityListener != nil {
		v.securityListener.OnClose(s)
	}
}

func (v *VirtualSecurityTCPSocket) hasWriteData() bool {
	return v.writeDataBuffer.Len() > 0
}
