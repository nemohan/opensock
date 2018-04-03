package utility
import (
	"sync/atomic"
)
type LogContext struct {
	id uint32
	*LogModule
	handle *LogModule
}

var logID uint32


//NewLogContext create a new log context, it will create a unique id when the argument id is 0
func NewLogContext(id uint32, log *LogModule) *LogContext {
	if id == 0{
		id = atomic.AddUint32(&logID, 1)
	}
	return &LogContext{id, log, log}
}

func (l *LogContext) GetID()uint32{
	return l.id
}


//LogDebug output debug level information
func (l *LogContext) LogDebug(format string, v ...interface{}) {
	l.Log(logLevelDebug|(l.id<<constLogLevelShift), format, v...)
}

//LogInfo output debug level information
func (l *LogContext) LogInfo(format string, v ...interface{}) {
	l.Log(logLevelInfo|(l.id<<constLogLevelShift), format, v...)
}

//LogWarn output warning level information
func (l *LogContext) LogWarn(format string, v ...interface{}) {
	l.Log(logLeveLWarn|(l.id<<constLogLevelShift), format, v...)
}

//LogFatal output debug level information
func (l *LogContext) LogFatal(format string, v ...interface{}) {
	l.Log(logLevelFatal|(l.id<<constLogLevelShift), format, v...)
}
