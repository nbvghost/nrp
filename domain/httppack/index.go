package httppack

import (
	"sync"
	"time"
)

type OutList struct {
	Data map[uint64]*HttpPack
	Lock sync.Mutex
}

func (ol *OutList) Set(key uint64, value *HttpPack) {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	if ol.Data == nil {
		ol.Data = make(map[uint64]*HttpPack)
	}
	ol.Data[key] = value
}
func (ol *OutList) Get(key uint64) (*HttpPack, bool) {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	if ol.Data == nil {
		ol.Data = make(map[uint64]*HttpPack)
	}
	_, ok := ol.Data[key]
	return ol.Data[key], ok
}
func (ol *OutList) Del(key uint64) {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	delete(ol.Data, key)
}

type HttpPack struct {
	Out  chan []byte
	Time time.Time
}
