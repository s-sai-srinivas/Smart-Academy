package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelFatal LogLevel = "FATAL"
	LevelPanic LogLevel = "PANIC"
)

// Field represents a structured log field
type Field struct {
	Value interface{}
	Key   string
}

// Logger is a structured logger with context support
type Logger struct {
	logger      *slog.Logger
	service     string
	environment string
	fields      []Field
	mu          sync.RWMutex
}

// Config holds logger configuration
type Config struct {
	Writer      io.Writer
	Service     string
	Environment string
	LogLevel    LogLevel
	EnableJSON  bool
}

// global logger instance
var (
	globalLogger *Logger
	once         sync.Once
)

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	var err error
	once.Do(func() {
		globalLogger, err = New(cfg)
	})
	return err
}

// New creates a new structured logger
func New(cfg Config) (*Logger, error) {
	if cfg.Service == "" {
		cfg.Service = "coding-platform"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Set minimum log level
	var level slog.Level
	switch cfg.LogLevel {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	if cfg.EnableJSON {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		logger:      logger,
		service:     cfg.Service,
		environment: cfg.Environment,
		fields:      make([]Field, 0),
	}, nil
}

// GetGlobal returns the global logger instance
func GetGlobal() *Logger {
	if globalLogger == nil {
		// Initialize with defaults if not initialized
		_ = Init(Config{EnableJSON: false})
	}
	return globalLogger
}

// WithFields adds fields to the logger context
func (l *Logger) WithFields(fields ...Field) *Logger {
	l.mu.RLock()
	existingFields := make([]Field, len(l.fields))
	copy(existingFields, l.fields)
	l.mu.RUnlock()

	newLogger := &Logger{
		logger:      l.logger,
		service:     l.service,
		environment: l.environment,
		fields:      append(existingFields, fields...),
	}
	return newLogger
}

// With adds a single field to the logger context
func (l *Logger) With(key string, value interface{}) *Logger {
	return l.WithFields(Field{Key: key, Value: value})
}

// buildArgs converts fields to slog args
func (l *Logger) buildArgs(additionalFields ...Field) []interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	args := make([]interface{}, 0, len(l.fields)+len(additionalFields)+4)

	// Add default fields
	args = append(args,
		"service", l.service,
		"environment", l.environment,
	)

	// Add context fields
	for _, f := range l.fields {
		args = append(args, f.Key, f.Value)
	}

	// Add additional fields
	for _, f := range additionalFields {
		args = append(args, f.Key, f.Value)
	}

	return args
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.buildArgs(fields...)...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.buildArgs(fields...)...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.buildArgs(fields...)...)
}

// Error logs an error message with stack trace
func (l *Logger) Error(msg string, err error, fields ...Field) {
	allFields := l.buildArgs(fields...)
	if err != nil {
		allFields = append(allFields, "error", err.Error())
		// Add stack trace for error context
		if _, file, line, ok := runtime.Caller(1); ok {
			allFields = append(allFields, "caller", fmt.Sprintf("%s:%d", file, line))
		}
	}
	l.logger.Error(msg, allFields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...Field) {
	allFields := l.buildArgs(fields...)
	if err != nil {
		allFields = append(allFields, "error", err.Error())
	}
	l.logger.Log(nil, slog.LevelError+4, msg, allFields...)
	os.Exit(1)
}

// Panic logs a panic message and panics
func (l *Logger) Panic(msg string, err error, fields ...Field) {
	allFields := l.buildArgs(fields...)
	if err != nil {
		allFields = append(allFields, "error", err.Error())
	}
	l.logger.Log(nil, slog.LevelError+4, msg, allFields...)
	panic(fmt.Sprintf("%s: %v", msg, err))
}

// LogRequest logs HTTP request details
func (l *Logger) LogRequest(method, path, clientIP string, statusCode int, latency time.Duration, userRegdNo string) {
	l.Info("HTTP request",
		Field{Key: "method", Value: method},
		Field{Key: "path", Value: path},
		Field{Key: "client_ip", Value: clientIP},
		Field{Key: "status_code", Value: statusCode},
		Field{Key: "latency_ms", Value: latency.Milliseconds()},
		Field{Key: "user", Value: userRegdNo},
	)
}

// LogPanic logs panic recovery with full stack trace
func (l *Logger) LogPanic(recoveredErr interface{}, stackTrace string) {
	l.Error("Panic recovered",
		fmt.Errorf("%v", recoveredErr),
		Field{Key: "stack_trace", Value: stackTrace},
	)
}

// LogErrorContext logs an error with additional context
func (l *Logger) LogErrorContext(operation string, err error, contextFields ...Field) {
	l.Error("Operation failed: "+operation, err, contextFields...)
}

// KV creates a key-value field pair
func KV(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// SafeJSON marshals data to JSON safely for logging
func SafeJSON(data interface{}) string {
	if data == nil {
		return "null"
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal: %v"}`, err)
	}
	return string(bytes)
}

// Redact creates a field with redacted value for sensitive data
func Redact(key string) Field {
	return Field{Key: key, Value: "[REDACTED]"}
}

// MaskEmail partially masks an email address for logging
func MaskEmail(email string) string {
	if len(email) < 5 {
		return "***"
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 0 {
		return "***"
	}
	return email[:2] + "***" + email[atIndex:]
}
