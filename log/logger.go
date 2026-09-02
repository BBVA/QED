package log

import (
	"io"
	"log"
	"strings"
	"sync"
)

// Level represents the logging level.
type Level uint32

const (
	// NotSet level is used to indicate that no level has been set
	// and allow for a default to be used
	NotSet Level = iota

	// Off is intended to avoid tracing any action.
	// This is the highest possible rank.
	Off

	// Fatal designates very severe errors that lead
	// the application to abort (e.g. data corruption, failures
	// to join the cluster...).
	Fatal

	// Error designates rare error events that might still allow
	// the application to continue running (e.g. loss of contact
	// with peers...).
	Error

	// Warn designates potentially harmful situations (e.g. encoding
	// errors, unknown message types...).
	Warn

	// Info designates informational messages that highlight the
	// main progress of the application at coarse-grained level.
	// It should be used to document main state changes in the application
	// (e.g. bootstrap, shutdown, main cluster operations...).
	Info

	// Debug designates fine-grained informational events that are
	// useful to debug the application. Don't use it in production.
	Debug

	// Trace designates even finer-grained informational events
	// than Debug level. Don't use it in production.
	Trace
)

// String returns a string representation of the level
func (l Level) String() string {
	switch l {
	case NotSet:
		return "unknown"
	case Off:
		return "off"
	case Fatal:
		return "fatal"
	case Error:
		return "error"
	case Warn:
		return "warn"
	case Info:
		return "info"
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	default:
		return "unknown"
	}
}

// LevelFromString returns a Level type for the named log level, or
// "NotSet" if the level passed as argument is invalid.
func LevelFromString(level string) Level {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "off":
		return Off
	case "fatal":
		return Fatal
	case "error":
		return Error
	case "warn":
		return Warn
	case "info":
		return Info
	case "debug":
		return Debug
	case "trace":
		return Trace
	default:
		return NotSet
	}
}

// Logger describes the interface that must be implemented
// by all loggers.
type Logger interface {
	// Trace emits a message at the TRACE level.
	Trace(msg string)
	// Tracef formats a message according to a format specifier
	// and emits it at the TRACE level.
	Tracef(format string, args ...interface{})
	// Debug emits a message at the DEBUG level.
	Debug(msg string)
	// Debugf formats a message according to a format specifier
	// and emits it at the DEBUG level.
	Debugf(format string, args ...interface{})
	// Info emits a message at the INFO level.
	Info(msg string)
	// Infof formats a message according to a format specifier
	// and emits it at the INFO level.
	Infof(format string, args ...interface{})
	// Warn emits a message at the WARN level.
	Warn(msg string)
	// Warnf formats a message according to a format specifier
	// and emits it at the WARN level.
	Warnf(format string, args ...interface{})
	// Error emits a message at the ERROR level.
	Error(msg string)
	// Errorf formats a message according to a format specifier
	// and emits it at the ERROR level.
	Errorf(format string, args ...interface{})
	// Fatal emits a message at the FATAL level
	// and exits the application (os.Exit(1)).
	Fatal(msg string)
	// Fatalf formats a message according to a format specifier,
	// emits it at the FATAL level and exits the application
	// (os.Exit(1)).
	Fatalf(format string, args ...interface{})
	// Panic emits a message at the FATAL level
	// and panics.
	Panic(msg string)
	// Panicf formats a message according to a format specifier,
	// emits it at the FATAL level and panics.
	Panicf(format string, args ...interface{})

	// Creates a logger that will prepend the given name on front of all
	// messages. If the logger has a previously set name, the new value
	// will be the appended to it.
	Named(name string) Logger

	// Creates a logger that will prepend the given name on front of all
	// messages. It overrides any previously set name.
	ResetNamed(name string) Logger

	// WithLevel creates a logger with the given level changed.
	WithLevel(level Level) Logger

	// StdLogger returns a logger implementation that conforms to the
	// stdlib log.Logger interface. This allows packages that expect
	// to be using the standard library log to actually use this logger.
	StdLogger(opts *StdLoggerOptions) *log.Logger

	// StdWriter returns a io.Writer implementation that conforms to
	// io.Writer, which can be passed into log.SetOutput().
	StdWriter(opts *StdLoggerOptions) io.Writer
}

// LoggerOptions can be used to configure a new logger.
type LoggerOptions struct {
	// Name of the subsystem to prefix logs with.
	Name string

	// Level is the threshold for the logger. Any log trace less
	// severe is suppressed.
	Level Level

	// Output is the writer implementation where to write logs to.
	// If nil, defaults to os.Stderr.
	Output io.Writer

	// TimeFormat is the time format to use instead of the default one.
	TimeFormat string

	// IncludeLocation includes file and line information in each log line.
	IncludeLocation bool

	// Mutex is an optional mutex pointer in case Output is shared.
	Mutex *sync.Mutex
}

// StdLoggerOptions can be used to configure a new standard logger.
type StdLoggerOptions struct {
	// Indicate that some minimal parsing should be done on strings to try
	// and detect their level and re-emit them.
	// This supports the strings like [FATAL], [ERROR], [TRACE], [WARN], [INFO],
	// [DEBUG] and strip it off before reapplying it.
	InferLevels bool

	// ForceLevel is used to force all output from the standard logger to be at
	// the specified level. Similar to InferLevels, this will strip any level
	// prefix contained in the logged string before applying the forced level.
	// If set, this overrides InferLevels.
	ForceLevel Level
}

// New returns a new logger configured with
// the given options.
func New(opts *LoggerOptions) Logger {
	if opts == nil {
		opts = &LoggerOptions{}
	}

	output := opts.Output
	if output == nil {
		output = DefaultOutput
	}

	level := opts.Level
	if level == NotSet {
		level = DefaultLevel
	}

	mutex := opts.Mutex
	if mutex == nil {
		mutex = new(sync.Mutex)
	}

	timeFormat := opts.TimeFormat
	if timeFormat == "" {
		timeFormat = DefaultTimeFormat
	}

	intLogger := internalLogger{
		name:       opts.Name,
		caller:     opts.IncludeLocation,
		timeFormat: opts.TimeFormat,
		level:      opts.Level,
		mutex:      mutex,
		writer:     newWriter(output),
	}

	var l Logger
	switch level {
	case Off:
		l = &silentLogger{intLogger}
	case Fatal:
		l = &fatalLogger{intLogger}
	case Error:
		l = &errorLogger{intLogger}
	case Warn:
		l = &warnLogger{intLogger}
	case Info:
		l = &infoLogger{intLogger}
	case Debug:
		l = &debugLogger{intLogger}
	case Trace:
		l = &traceLogger{intLogger}
	default:
		l = &infoLogger{intLogger}
	}

	return l
}
