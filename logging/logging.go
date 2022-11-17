package logging

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"cloud.google.com/go/logging"
	"golang.org/x/net/context"

	"github.com/paper-trade-chatbot/be-order/config"
)

// Logger is our logger instance abstraction.
type Logger struct {
	*logging.Logger
}

const (
	ContextKeyAccount   = "account"
	ContextKeyRequestId = "request_id"
	ContextKeyUserId    = "user_id"
)

// Singleton StackDriver client and logger instances.
var stackDriverClient *logging.Client
var stackDriverLogger = &Logger{}

// Static configuration variables initalized at runtime.
var logLevel uint
var stackDriverEnabled bool
var gRPCConnectTimeout time.Duration
var projectID string

// Log levels.
const (
	logLevelFirst = iota
	logLevelCritical
	logLevelError
	logLevelWarn
	logLevelInfo
	logLevelDebug
	logLevelLast
)

// Log level to label string.
var logLabels = []string{
	"",
	"\x1b[0;37;41m  CRIT \x1b[m",
	"\x1b[0;30;41m ERROR \x1b[m",
	"\x1b[0;30;43m  WARN \x1b[m",
	"\x1b[0;30;47m  INFO \x1b[m",
	"\x1b[0;30;42m DEBUG \x1b[m",
	"",
}

// Log level to StackDriver severity.
var logSeverities = []logging.Severity{
	logging.Default,
	logging.Critical,
	logging.Error,
	logging.Warning,
	logging.Info,
	logging.Debug,
	logging.Default,
}

// init loads the logging configurations.
func init() {
	logLevel = config.GetUint("LOG_LEVEL")
	stackDriverEnabled = config.GetBool("STACKDRIVER_ENABLED")
	gRPCConnectTimeout = config.GetMilliseconds("GRPC_CONNECT_TIMEOUT_MS")
	projectID = config.GetString("PROJECT_ID")
}

// Initialize initializes the logger module.
func Initialize(ctx context.Context) {
	// Do not setup StackDriver client if not configured.
	if !stackDriverEnabled {
		return
	}

	// Setup timeout context for connecting to StackDriver.
	timeoutCtx, cancel := context.WithTimeout(ctx, gRPCConnectTimeout)
	defer cancel()

	// Create StackDriver logger client.
	var err error
	stackDriverClient, err = logging.NewClient(timeoutCtx, projectID)
	if err != nil {
		panic(err)
	}

	// Check StackDriver connection.
	if err = stackDriverClient.Ping(timeoutCtx); err != nil {
		panic(err)
	}
	// Create StackDriver logger instance.
	stackDriverLogger = &Logger{stackDriverClient.Logger(projectID)}
}

// Finalize finalizes the logging module.
func Finalize() {
	// Check if client and logger are valid.
	if stackDriverClient == nil || stackDriverLogger == nil {
		return
	}

	// Flush logs and properly close logging service connection.
	if err := stackDriverClient.Close(); err != nil {
		now := float64(time.Now().UnixNano()) / float64(time.Second)
		fmt.Fprintf(os.Stderr,
			"\r\x1b[100m%f\x1b[m %s\x1b[m \x1b[100m%12s\x1b[m %s\n",
			now, logLabels[logLevelError], projectID, err.Error())
	}
}

// NewLogger returns a new copy of a logger instance.
func NewLogger() (*Logger, error) {
	// Check if StackDriver client has been initialized.
	var logger *logging.Logger
	if stackDriverClient != nil {
		logger = stackDriverClient.Logger(projectID)
	}

	// Create and return new logger instance.
	return &Logger{logger}, nil
}

// Critical logs a message of critical severity.
func Critical(requestCtx context.Context, format string, args ...interface{}) {
	logWithLineNumber(requestCtx, logLevelCritical, format, args...)
}

// Critical logs a message of critical severity using the given logger.
func (logger *Logger) Critical(requestCtx context.Context, format string, args ...interface{}) {
	logWithLineNumber(requestCtx, logLevelCritical, format, args...)
}

// Error logs a message of error severity.
func Error(requestCtx context.Context, format string, args ...interface{}) {
	logWithLineNumber(requestCtx, logLevelError, format, args...)
}

// Error logs a message of error severity using the given logger.
func (logger *Logger) Error(requestCtx context.Context, format string, args ...interface{}) {
	logWithLineNumber(requestCtx, logLevelError, format, args...)
}

// Warn logs a message of warning severity.
func Warn(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelWarn, format, args...)
}

// Warn logs a message of warning severity using the given logger.
func (logger *Logger) Warn(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelWarn, format, args...)
}

// Info logs a message of informational severity.
func Info(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelInfo, format, args...)
}

// Info logs a message of informational severity using the given logger.
func (logger *Logger) Info(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelInfo, format, args...)
}

// Debug logs a message of debugging severity.
func Debug(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelDebug, format, args...)
}

// Debug logs a message of debugging severity using the given logger.
func (logger *Logger) Debug(requestCtx context.Context, format string, args ...interface{}) {
	log(requestCtx, logLevelDebug, format, args...)
}

// log is the general logging utility function used by all log levels.
func log(requestCtx context.Context, level uint, format string, args ...interface{}) {
	// Perform logging only if configured above and within valid log level.
	if level <= logLevelFirst || level >= logLevelLast || level > logLevel {
		return
	}

	// Compose log message.
	message := fmt.Sprintf(format, args...)

	requestId, _ := requestCtx.Value(ContextKeyRequestId).(string)
	account, _ := requestCtx.Value(ContextKeyAccount).(string)

	// Log to StackDriver logging service
	if stackDriverClient != nil && stackDriverLogger != nil {
		stackDriverLogger.Log(logging.Entry{
			Severity: logSeverities[level],
			Payload:  message,
			Labels: map[string]string{
				"request_id": requestId,
				"account":    account,
			},
		})
	}

	// now is the current Unix timestamp in floating point.
	// now := float64(time.Now().UnixNano()) / float64(time.Second)
	nowInString := time.Now().Format("2006-01-02 15:04:05.000")

	// Reset terminal color.
	fmt.Print("\x1b[m")

	// Log to standard output.
	fmt.Fprintf(os.Stdout,
		"\r\x1b[100m%s\x1b[m %s\x1b[m \x1b[100m%12s\x1b[m %s\n",
		nowInString, logLabels[level], requestId, message)
}

// logWithLineNumber performs usual logging but with an extra line number arg.
func logWithLineNumber(requestCtx context.Context, level uint, format string,
	args ...interface{}) {
	// Get caller file name and line number.
	_, filepath, line, ok := runtime.Caller(2)
	if ok {
		filename := path.Base(filepath)
		format = fmt.Sprintf("%s (%s:%d)", format, filename, line)
	}
	log(requestCtx, level, format, args...)
}
