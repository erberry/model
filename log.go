package model

import "fmt"

type logger interface {
	Info(msg string)
	Error(msg string)
}

type fmtLogger struct {
}

var (
	log logger
)

func init() {
	SetLogger(fmtLogger{})
}

// SetLogger SetLogger
func SetLogger(l logger) {
	log = l
}

func (fmtLogger) Info(msg string) {
	fmt.Println(msg)
}
func (fmtLogger) Error(msg string) {
	fmt.Println(msg)
}
