package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var Logger *log.Logger

const flag = log.Ldate | log.Ltime

func init() {
	Logger = log.New(os.Stdout, "", flag)
}

func GetLogger() *log.Logger {
	return Logger
}

func fileName(dir string) string {
	return dir + time.Now().Format("2006-01-02") + "-server.log"
}

func SetOutput(dir string) error {
	logFile, err := os.OpenFile(fileName(dir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return fmt.Errorf("specified log file path does not exist: %v", dir)
	}
	writer := io.MultiWriter(os.Stdout, logFile)

	Logger = log.New(writer, "", flag)
	return nil
}
