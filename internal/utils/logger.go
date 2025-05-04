package utils

import (
	"fmt"
	"log"
	"runtime"
)

type level int

const (
	DEBUG level = iota
	INFO
	WARN
	ERROR
)

func (l level) effect() int {
	switch l {
	case DEBUG:
		return 0
	case INFO:
		return 2
	case WARN:
		return 5
	case ERROR:
		return 1
	default:
		AssertFail("Invalid log level")
		return 0
	}
}

func (l level) color() int {
	switch l {
	case DEBUG:
		return 92
	case INFO:
		return 37
	case WARN:
		return 93
	case ERROR:
		return 91
	default:
		AssertFail("Invalid log level")
		return 0
	}
}

func (l level) toStr() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		AssertFail("Invalid log level")
		return ""
	}
}

type logMsg struct {
	l    level
	args []any
	file string
	line int
}

type logger struct {
	c chan logMsg
}

var gLogger *logger = nil

func InitLogger() {
	if gLogger != nil {
		return
	}
	gLogger = &logger{c: make(chan logMsg)}
	go func() {
		for msg := range gLogger.c {
			msgHandle(msg)
		}
	}()
}

func LogDebug(v ...any) {
	InitLogger()
	_, file, line, _ := runtime.Caller(1)
	gLogger.c <- logMsg{args: v, l: DEBUG, file: file, line: line}
}

func LogInfo(v ...any) {
	InitLogger()
	_, file, line, _ := runtime.Caller(1)
	gLogger.c <- logMsg{args: v, l: INFO, file: file, line: line}
}

func LogWarn(v ...any) {
	InitLogger()
	_, file, line, _ := runtime.Caller(1)
	gLogger.c <- logMsg{args: v, l: WARN, file: file, line: line}
}

func LogError(v ...any) {
	InitLogger()
	_, file, line, _ := runtime.Caller(1)
	gLogger.c <- logMsg{args: v, l: ERROR, file: file, line: line}
}

func msgHandle(msg logMsg) {
	buf := fmt.Sprintf("\033[2;m %s:%d\033[0;m", msg.file, msg.line)
	buf += fmt.Sprintf("\033[0;%dm [%s]\033[0;m ", msg.l.color(), msg.l.toStr())
	buf += fmt.Sprintf("\033[%d;%dm", msg.l.effect(), msg.l.color())
	buf += fmt.Sprint(msg.args...)
	buf += fmt.Sprint("\033[0;m")
	log.Println(buf)
}
