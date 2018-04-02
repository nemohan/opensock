package utility

type LogContext struct {
	id uint32
	*LogModule
	handle *LogModule
}

//NewLogContext create a new log context
func NewLogContext(id uint32, log *LogModule) *LogContext {
	return &LogContext{id, log, log}
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
