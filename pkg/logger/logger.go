package logger

import (
	"fmt"
	"log"
	"os"
)

func Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format+"\n", args...)
}

func Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format+"\n", args...)
}

func Debug(format string, args ...interface{}) {
	if os.Getenv("GIN_MODE") != "release" {
		log.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func Fatal(format string, args ...interface{}) {
	log.Printf("[FATAL] "+format+"\n", args...)
	os.Exit(1)
}

func Printf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}