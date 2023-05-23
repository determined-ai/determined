package tasklogger

var taskLogger *Logger

func Init(w Writer) {
	taskLogger = New(w)
}

func Get() *Logger {
	if taskLogger == nil {
		panic("tasklogger uninitialized; Get called before Init")
	}
	return taskLogger
}
