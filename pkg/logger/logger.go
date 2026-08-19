package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	infoLog  = log.New(os.Stdout, "\033[34m[INFO]\033[0m ", log.LstdFlags|log.Lmsgprefix)
	warnLog  = log.New(os.Stdout, "\033[33m[WARN]\033[0m ", log.LstdFlags|log.Lmsgprefix)
	errorLog = log.New(os.Stderr, "\033[31m[ERROR]\033[0m ", log.LstdFlags|log.Lmsgprefix)
	botLog   = log.New(os.Stdout, "\033[32m[BOT]\033[0m ", log.LstdFlags|log.Lmsgprefix)
	apiLog   = log.New(os.Stdout, "\033[36m[API]\033[0m ", log.LstdFlags|log.Lmsgprefix)
	dbLog    = log.New(os.Stdout, "\033[35m[DB]\033[0m ", log.LstdFlags|log.Lmsgprefix)
)

func Info(module string, format string, v ...interface{}) {
	infoLog.Printf("[%s] %s", module, fmt.Sprintf(format, v...))
}

func Warn(module string, format string, v ...interface{}) {
	warnLog.Printf("[%s] %s", module, fmt.Sprintf(format, v...))
}

func Error(module string, format string, v ...interface{}) {
	errorLog.Printf("[%s] %s", module, fmt.Sprintf(format, v...))
}

func Bot(format string, v ...interface{}) {
	botLog.Printf(format, v...)
}

func API(format string, v ...interface{}) {
	apiLog.Printf(format, v...)
}

func DB(format string, v ...interface{}) {
	dbLog.Printf(format, v...)
}
