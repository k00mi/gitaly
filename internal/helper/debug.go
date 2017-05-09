package helper

import (
	"log"
	"os"
)

func Debugf(format string, args ...interface{}) {
	if os.Getenv("GITALY_DEBUG") != "1" {
		return
	}

	log.Printf("debug: "+format, args...)
}
