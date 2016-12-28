package socket

import (
	"container/list"
	"sync"
)

const kMaxBufferCount = 10

type bufferPool struct {
	buffer map[int]*list.List
	mutex  sync.Mutex
}

var bufferPoolOnce sync.Once
var bufferPoolInstance *bufferPool = nil

func GetBufferPool() *bufferPool {
	bufferPoolOnce.Do(func() {
		bufferPoolInstance = new(bufferPool)
		bufferPoolInstance.buffer = make(map[int]*list.List)
	})
	return bufferPoolInstance
}

func (b *bufferPool) GetBuffer(bufferSize int) *[]byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	var bufferList *list.List = nil
	var ok bool = false
	if bufferList, ok = b.buffer[bufferSize]; !ok {
		bufferList = list.New()
		b.buffer[bufferSize] = bufferList
	}
	var retBuffer *[]byte = nil
	if bufferList.Len() == 0 {
		buf := make([]byte, bufferSize)
		retBuffer = &buf
	} else {
		bufValue := bufferList.Front()
		retBuffer = bufValue.Value.(*[]byte)
		bufferList.Remove(bufValue)
	}
	return retBuffer
}

func (b *bufferPool) PutBuffer(data *[]byte) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	var bufferSize = len(*data)
	var bufferList *list.List = nil
	var ok bool = false
	if bufferList, ok = b.buffer[bufferSize]; !ok {
		bufferList = list.New()
		b.buffer[bufferSize] = bufferList
	}
	if bufferList.Len() >= kMaxBufferCount {
		return
	}
}

func (b *bufferPool) balance() {
	// TODO: balance all size buffer
}
