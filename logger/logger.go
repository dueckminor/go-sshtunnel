package logger

import (
	"log"
	"os"
)

// L is the global logger
var L = log.New(os.Stdout, "sshtunnel: ", log.Lshortfile|log.LstdFlags)
