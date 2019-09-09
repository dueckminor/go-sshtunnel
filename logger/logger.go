package logger

import (
	"log"
	"os"
)

var L = log.New(os.Stdout, "sshuttle: ", log.Lshortfile|log.LstdFlags)
