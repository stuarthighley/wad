package wad

import (
	"io"
	"log"
)

var logger *log.Logger = log.New(io.Discard, "", log.LstdFlags)

func SetLogger(l *log.Logger) {
	logger = l
}
