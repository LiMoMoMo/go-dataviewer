package main

import (
	"context"
	"math/rand"

	viewer "github.com/LiMoMoMo/go-dataviewer"
)

var tempMap map[string]int

type Demo struct {
	viewer.Viewer
	t int32
	w float32
}

func (b *Demo) GetVal(name string) interface{} {
	val, ok := tempMap[name]
	if !ok {
		tempMap[name] = 0
	}
	tempMap[name] = val + rand.Intn(100)
	switch name {
	case "Value":
		return int64(rand.Intn(100))
	case "Memory":
		return int64(rand.Intn(1024 * 1024))
	case "BandWidth":
		return int64(rand.Intn(1024 * 1024))
	case "Rank":
		return tempMap
	}
	return ""
}

func main() {
	tempMap = make(map[string]int)
	ctx, _ := context.WithCancel(context.Background())
	demo := Demo{}
	demo.SetChild(&demo, ctx)
	demo.Register("Value", viewer.TypeValue)
	demo.Register("Memory", viewer.TypeMemory)
	demo.Register("BandWidth", viewer.TypeBandWidth)
	demo.Register("Rank", viewer.TypeRank)
	demo.SetHttp("0.0.0.0:8946", 1)
	demo.Run()
	select {}
}
