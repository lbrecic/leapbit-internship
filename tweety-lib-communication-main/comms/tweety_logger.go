package comms

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	CLEAN   = 0
	INFO    = 1
	WARNING = 2
	ERROR   = 3
	DEBUG   = 4
	space   = "\n                             "
)

var (
	typeMap map[int64]string
)

func init() {
	typeMap = make(map[int64]string)
	typeMap[CLEAN] = "CLEAN"
	typeMap[INFO] = "INFO"
	typeMap[WARNING] = "WARNING"
	typeMap[ERROR] = "ERROR"
	typeMap[DEBUG] = "DEBUG"
}

type TweetyLogger struct {
	Level    int64
	FileName string
	FilePath string
	File     *os.File
}

func NewTweetyLogger(fileName string, filePath string, level int64) *TweetyLogger {
	logger := &TweetyLogger{
		Level:    level,
		FileName: fileName,
		FilePath: filePath,
	}
	var err error
	logger.File, err = os.Create(fmt.Sprintf("%s%s%s", filePath, fileName, ".txt"))
	if err != nil {
		TweetyLog(ERROR, "Log file can't open. error: %s", err)
	}
	return logger
}

func (logger *TweetyLogger) LogData(logType int64, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	var logMsg string
	switch logType {
	case CLEAN:
		logMsg = fmt.Sprintf("%s\n", msg)
	case INFO, ERROR, WARNING, DEBUG:
		logMsg = fmt.Sprintf("[%s]: %s\n", typeMap[logType], msg)
	default:
		logMsg = fmt.Sprintf("[INVALID]: Invalid code!!! Message: %s\n", msg)
	}
	log.Print(logMsg)
	timedMsg := fmt.Sprintf("%s %s", time.Now().Format(time.ANSIC), logMsg)
	if logType <= logger.Level {
		logData := []byte(timedMsg)
		_, err := logger.File.Write(logData)
		if err != nil {
			TweetyLog(ERROR, "Logger - File error: %s", err)
		}
	}
}

func TweetyLog(logType int64, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	switch logType {
	case CLEAN:
		log.Printf("%s\n", msg)
	case INFO, ERROR, WARNING, DEBUG:
		log.Printf("[%s]: %s\n", typeMap[logType], msg)
	default:
		log.Printf("[INVALID]: Invalid code!!! Message: %s\n", msg)
	}
}
