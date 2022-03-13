package main

import (
	"bytes"
	"io"
)

//go:generate go run ../main.go -testsignature=Greet
func Greet(name string) (greeting string) {
	return "hello " + name + "\n"
}

//go:generate go run ../main.go -testsignature=Dismiss
func Dismiss(name string) (greeting string) {
	return "goodbye " + name + "\n"
}

//go:generate go run ../main.go -testsignature=AddInts
func AddInts(a, b int) (sum int) {
	return a + b
}

//go:generate go run ../main.go -testsignature=AddInt8
func AddInt8(a, b int8) (sum int8) {
	return a + b
}

//go:generate go run ../main.go -testsignature=AddInt16
func AddInt16(a, b int16) (sum int16) {
	return a + b
}

//go:generate go run ../main.go -testsignature=AddInt32
func AddInt32(a, b int32) (sum int32) {
	return a + b
}

//go:generate go run ../main.go -testsignature=AddInt64
func AddInt64(a, b int64) (sum int64) {
	return a + b
}

//go:generate go run ../main.go -testsignature=EchoFloat32
func EchoFloat32(a float32) (e float32) {
	return a
}

//go:generate go run ../main.go -testsignature=EchoFloat64
func EchoFloat64(a float64) (e float64) {
	return a
}

//go:generate go run ../main.go -testsignature=BoolFlag
func BoolFlag(on bool) (position string) {
	if on {
		return "on"
	}
	return "off"
}

//go:generate go run ../main.go -testsignature=StdInOut
func StdInOut(in io.Reader) (out io.Reader) {
	b, err := io.ReadAll(in)
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(bytes.ToUpper(b))
}
