package dataviewer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// we should keep the records until send to browser.
var sampleList map[string]interface{}

func init() {
	sampleList = make(map[string]interface{})
}

type Event interface {
	handle()
	setBase(v *Viewer)
}

type baseEvent struct {
	v *Viewer
}

func (e *baseEvent) setBase(v *Viewer) {
	e.v = v
}

// the writeEvent
type writeEvent struct {
	baseEvent
	s sample
}

func (e *writeEvent) handle() {
	timeList, ok := sampleList["timestamp"]
	if !ok {
		sampleList["timestamp"] = []string{e.s.timeStamp}
	} else {
		ok := timeList.([]string)
		ok = append(ok, e.s.timeStamp)
		sampleList["timestamp"] = ok
	}
	for _, pair := range e.s.pairs {
		keylist, ok := sampleList[pair.key]
		if !ok {
			sampleList[pair.key] = []interface{}{pair.value}
		} else {
			ok := keylist.([]interface{})
			ok = append(ok, pair.value)
			sampleList[pair.key] = ok
		}
	}
}

// the readEvent
type readEvent struct {
	baseEvent
	callbackWriter http.ResponseWriter
	wg             *sync.WaitGroup
}

func (e *readEvent) handle() {
	defer e.wg.Done()
	// do read.
	// e.callbackWriter.Write(data)
	data, err := json.Marshal(sampleList)
	if err != nil {
		fmt.Println("Viewer readEvent Marshal error", err)
	}
	e.callbackWriter.Write(data)
	// clear map.
	for key, _ := range sampleList {
		delete(sampleList, key)
	}
}

// the typeEvent
type typeEvent struct {
	baseEvent
	callbackWriter http.ResponseWriter
	wg             *sync.WaitGroup
}

func (e *typeEvent) handle() {
	defer e.wg.Done()
	// return types.
	data, err := json.Marshal(e.v.types)
	if err != nil {
		fmt.Println("Viewer typeEvent Marshal error", err)
	}
	e.callbackWriter.Write(data)
}
