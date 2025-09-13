package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type MultiWriter struct {
	writers []io.Writer
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		if _, err := w.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (mw *MultiWriter) Add(writer io.Writer) {
	mw.writers = append(mw.writers, writer)
}

type ColoredLogger struct {
	verbose bool
	mu      sync.RWMutex
	writers map[LogLevel]io.Writer
	loggers map[LogLevel]*log.Logger
}

var globalLogger *ColoredLogger

func init() {
	globalLogger = &ColoredLogger{
		verbose: false,
		writers: make(map[LogLevel]io.Writer),
		loggers: make(map[LogLevel]*log.Logger),
	}

	for level := DEBUG; level <= FATAL; level++ {
		globalLogger.writers[level] = os.Stdout
		globalLogger.loggers[level] = log.New(os.Stdout, "", 0)
	}
}

func SetVerbose(verbose bool) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	globalLogger.verbose = verbose
}

func IsVerbose() bool {
	globalLogger.mu.RLock()
	defer globalLogger.mu.RUnlock()
	return globalLogger.verbose
}

func SetWriter(level LogLevel, writer io.Writer) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	globalLogger.writers[level] = writer
	globalLogger.loggers[level] = log.New(writer, "", 0)
}

func SetWriterForAll(writer io.Writer) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	for level := DEBUG; level <= FATAL; level++ {
		globalLogger.writers[level] = writer
		globalLogger.loggers[level] = log.New(writer, "", 0)
	}
}

func AddWriter(level LogLevel, writer io.Writer) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	currentWriter := globalLogger.writers[level]

	if mw, ok := currentWriter.(*MultiWriter); ok {
		mw.Add(writer)
	} else {
		multiWriter := NewMultiWriter(currentWriter, writer)
		globalLogger.writers[level] = multiWriter
		globalLogger.loggers[level] = log.New(multiWriter, "", 0)
	}
}

func AddWriterForAll(writer io.Writer) {
	for level := DEBUG; level <= FATAL; level++ {
		AddWriter(level, writer)
	}
}

func SetErrorWriter() {
	SetWriter(ERROR, os.Stderr)
	SetWriter(FATAL, os.Stderr)
}

func (cl *ColoredLogger) getColor(level LogLevel) string {
	switch level {
	case DEBUG:
		return ColorGray
	case INFO:
		return ColorBlue
	case WARN:
		return ColorYellow
	case ERROR:
		return ColorRed
	case FATAL:
		return ColorPurple
	default:
		return ColorWhite
	}
}

func (cl *ColoredLogger) formatMessage(level LogLevel, message string) string {
	timestamp := time.Now().Format("06-01-02 15:04:05")

	tsColor := ColorGray
	bracketColor := ColorGray
	levelColor := cl.getColor(level)
	reset := ColorReset

	return fmt.Sprintf(
		"%s[%s%s%s]%s %s%-5s%s %s%s",
		bracketColor, tsColor, timestamp, bracketColor, reset,
		levelColor, level.String(), reset,
		message, reset,
	)
}

func (cl *ColoredLogger) log(level LogLevel, format string, args ...interface{}) {
	cl.mu.RLock()
	if level == DEBUG && !cl.verbose {
		cl.mu.RUnlock()
		return
	}

	logger := cl.loggers[level]
	cl.mu.RUnlock()

	message := fmt.Sprintf(format, args...)
	formattedMessage := cl.formatMessage(level, message)

	logger.Println(formattedMessage)

	if level == FATAL {
		os.Exit(1)
	}
}

func Debug(format string, args ...interface{}) {
	globalLogger.log(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	globalLogger.log(INFO, format, args...)
}

func Warn(format string, args ...interface{}) {
	globalLogger.log(WARN, format, args...)
}

func Error(format string, args ...interface{}) {
	globalLogger.log(ERROR, format, args...)
}

func Fatal(format string, args ...interface{}) {
	globalLogger.log(FATAL, format, args...)
}

func GetLogFromLevel(level LogLevel) func(format string, args ...interface{}) {
	return func(format string, args ...interface{}) {
		globalLogger.log(level, format, args...)
	}
}
