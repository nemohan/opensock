package utility

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogModule struct {
	log              *log.Logger
	lock             *sync.Mutex
	msgChan          chan *logMsg
	exitChan         chan bool
	level            uint32
	file             *os.File
	logFile          string
	lineNumber       int
	limit            int
	signalReopenChan chan int
	path             string
}

type logMsg struct {
	level uint32
	data  string
}

/*
//LogContext contains goroutine id for debug purpose
type LogContext struct {
	id uint32
	*LogModule
	handle *LogModule
}
*/

const (
	logLevelDebug uint32 = iota
	logLevelInfo
	logLevelNotice
	logLeveLWarn
	logLevelErr
	logLevelFatal
)

const maxLogLineNumber = 500000
const (
	constLogLevelShift uint32 = 3
	constLogLevelMask  = 0x7
)

var logLevelTable = [6]string{"DBG",
	"INFO",
	"NOTICE",
	"WARN",
	"ERR",
	"FATAL"}


//NewLog create a log module instance
func NewLog(logFile string, logLevel string, limit int, path string) *LogModule {
	if len(path) == 0 {
		path = "./"
	}
	newLogFile := Format("%s/%s.log", path, logFile)
	file, err := os.Create(newLogFile)
	if err != nil {
		fmt.Printf("Failed to open log file:%s %s\n", logFile, err.Error())
		os.Exit(1)
	}

	level := logLevelInfo
	for l, e := range logLevelTable {
		if strings.Compare(e, logLevel) == 0 {
			level = uint32(l)
			break
		}
	}
	if limit <= maxLogLineNumber {
		limit = maxLogLineNumber
	}
	logger := log.New(file, "chat", log.Ldate|log.Lmicroseconds)
	log := &LogModule{
		log:              logger,
		lock:             new(sync.Mutex),
		msgChan:          make(chan *logMsg, 1024),
		exitChan:         make(chan bool, 1),
		level:            level,
		file:             file,
		limit:            limit,
		lineNumber:       0,
		logFile:          logFile,
		signalReopenChan: make(chan int, 1),
		path:             path,
	}
	go log.output()
	log.Log(level, "log file is:%s level:%s limit:%d line", logFile, logLevel, limit)
	return log
}


//GetHandle return the raw log handle
func (l *LogContext) GetHandle() *LogModule {
	return l.handle
}

//CatchPanic catch panic event and dump stack information
func CatchPanic(l *LogContext, onExit func()) {
	r := recover()
	if onExit != nil {
		onExit()
	}
	if r == nil {
		return
	}
	stack := make([]byte, 1<<16)
	size := runtime.Stack(stack, false)
	l.LogWarn("runtime fatal error:%v", r)
	l.LogWarn("stack begin***************")
	l.LogWarn("sack:%s", string(stack[:size]))
	l.LogWarn("stack end *****************")
}

//Log dump log information to file
func (l *LogModule) Log(level uint32, format string, v ...interface{}) {
	rawLevel := level & constLogLevelMask
	var file string
	var line int
	if level <= logLevelFatal {
		_, file, line, _ = runtime.Caller(2)
	} else {
		_, file, line, _ = runtime.Caller(2)
	}
	name := path.Base(file)
	newFormat := fmt.Sprintf("[%09d][%s]: %s  \nfile:%s line:%d\n", level>>constLogLevelShift, logLevelTable[rawLevel], format, name, line)

	msg := &logMsg{rawLevel, fmt.Sprintf(newFormat, v...)}
	l.msgChan <- msg
}

//LogExit notify log module exit
func (l *LogModule) Exit() {
	l.exitChan <- true
}

//SendSignal notify log some event
func (l *LogModule) SendSignal() {
	l.signalReopenChan <- 0
}

func (l *LogModule) openLogFile() {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, _ := now.Clock()
	logFile := Format("%s/%s_%d%d%d%d%d.log", l.path, l.logFile, year, month, day, hour, min)
	file, err := os.Create(logFile)
	if err != nil {
		l.log.Printf("Failed to create log file:%s err:%s\n", logFile, err.Error())
		return
	}
	l.file.Close()
	l.file = file
	l.lineNumber = 0
	l.log = log.New(file, "chat", log.Ldate|log.Lmicroseconds)
}

func (c *LogModule) output() {
	defer func() {
		if r := recover(); r != nil {
			c.log.Printf("runtime error:%v\n", r)
			stack := make([]byte, 1<<16)
			size := runtime.Stack(stack, false)
			c.log.Printf("runtime fatal error:%v\n", r)
			c.log.Printf("stack begin***************\n")
			c.log.Printf("sack:%s\n", string(stack[:size]))
			c.log.Printf("stack end *****************\n")
		}
	}()
	for {
		select {
		case <-c.exitChan:
			c.log.Printf("log module exit cleanly")
			c.file.Close()
			return
		case msg := <-c.msgChan:
			c.lineNumber += 2
			if msg.level == logLevelFatal {
				c.log.Fatal(msg.data)
			} else if msg.level >= c.level {
				c.log.Printf(msg.data)
			}
			if c.lineNumber < c.limit {
				continue
			}
			c.openLogFile()
		case <-c.signalReopenChan:
			c.log.Printf("got reopen signal. create new log file")
			c.openLogFile()
		}

	}

}
