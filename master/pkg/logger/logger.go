package logger

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
)

// New returns an echo logger connected to logrus.
func New() echo.Logger {
	return &logger{
		log: logrus.StandardLogger(),
	}
}

type logger struct {
	log *logrus.Logger
}

func mustMarshal(j log.JSON) string {
	b, err := json.MarshalIndent(j, "", "    ")
	if err != nil {
		panic(fmt.Sprintf("unable to parse log message: %v", j))
	}
	return "TEST" + string(b)
}

func (l *logger) SetLevel(v log.Lvl) { /* The logging level is set by the caller. */ }
func (l *logger) Level() log.Lvl {
	switch l.log.Level {
	case logrus.DebugLevel:
		return log.DEBUG
	case logrus.InfoLevel:
		return log.INFO
	case logrus.WarnLevel:
		return log.WARN
	case logrus.ErrorLevel:
		return log.ERROR
	case logrus.FatalLevel:
		return log.ERROR
	case logrus.PanicLevel:
		return log.ERROR
	default:
		panic(fmt.Sprintf("unexpected log level: %v", l.log.Level))
	}
}

func (l *logger) SetOutput(w io.Writer) { l.log.Out = w }
func (l *logger) Output() io.Writer     { return l.log.Out }

func (l *logger) SetPrefix(p string) { /* Logrus uses formatters rather than prefixes. */ }
func (l *logger) Prefix() string     { return "" }

func (l *logger) SetHeader(h string) { /* Logrus uses formatters rather than headers. */ }

func (l *logger) Print(i ...interface{})                    { l.log.Print(i...) }
func (l *logger) Printf(format string, args ...interface{}) { l.log.Printf(format, args...) }
func (l *logger) Printj(j log.JSON)                         { l.log.Println(mustMarshal(j)) }
func (l *logger) Debug(i ...interface{})                    { l.log.Debug(i...) }
func (l *logger) Debugf(format string, args ...interface{}) { l.log.Debugf(format, args...) }
func (l *logger) Debugj(j log.JSON)                         { l.log.Debugln(mustMarshal(j)) }
func (l *logger) Info(i ...interface{})                     { l.log.Info(i...) }
func (l *logger) Infof(format string, args ...interface{})  { l.log.Infof(format, args...) }
func (l *logger) Infoj(j log.JSON)                          { l.log.Infoln(mustMarshal(j)) }
func (l *logger) Warn(i ...interface{})                     { l.log.Warn(i...) }
func (l *logger) Warnf(format string, args ...interface{})  { l.log.Warnf(format, args...) }
func (l *logger) Warnj(j log.JSON)                          { l.log.Warnln(mustMarshal(j)) }
func (l *logger) Error(i ...interface{})                    { l.log.Error(i...) }
func (l *logger) Errorf(format string, args ...interface{}) { l.log.Errorf(format, args...) }
func (l *logger) Errorj(j log.JSON)                         { l.log.Errorln(mustMarshal(j)) }
func (l *logger) Fatal(i ...interface{})                    { l.log.Fatal(i...) }
func (l *logger) Fatalf(format string, args ...interface{}) { l.log.Fatalf(format, args...) }
func (l *logger) Fatalj(j log.JSON)                         { l.log.Fatalln(mustMarshal(j)) }
func (l *logger) Panic(i ...interface{})                    { l.log.Panic(i...) }
func (l *logger) Panicf(format string, args ...interface{}) { l.log.Panicf(format, args...) }
func (l *logger) Panicj(j log.JSON)                         { l.log.Panicln(mustMarshal(j)) }
