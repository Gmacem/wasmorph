package main

import (
	"github.com/extism/go-pdk"
)

//go:export TransformWrapper
func TransformWrapper() int32 {
	inputBytes := pdk.Input()
	pdk.Output(inputBytes)
	return 0
}

func main() {
	select {}
}
