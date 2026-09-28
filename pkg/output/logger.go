package output

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"sync"
	"time"
)

// Level mirrors slog.Level's severity ordering (higher is more severe) and
// adds FatalLevel/PanicLevel above ErrorLevel for messages that exit or
// panic after being logged, matching the levels DDEV relied on from logrus.
type Level slog.Level

const (
	DebugLevel = Level(slog.LevelDebug)
	InfoLevel  = Level(slog.LevelInfo)
	WarnLevel  = Level(slog.LevelWarn)
	ErrorLevel = Level(slog.LevelError)
	FatalLevel = Level(slog.LevelError + 4)
	PanicLevel = Level(slog.LevelError + 8)
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	default:
		return slog.Level(l).String()
	}
}

// Fields carries structured data attached via WithField/WithFields, rendered
// by a Formatter alongside the log message.
type Fields map[string]any

// Entry is a single formatted log record, handed to a Formatter, and the
// value WithField/WithFields return for chaining a message onto it.
type Entry struct {
	Logger  *Logger
	Time    time.Time
	Level   Level
	Message string
	Data    Fields
}

func (e *Entry) Print(args ...any) { e.Logger.emit(InfoLevel, e.Data, fmt.Sprint(args...)) }
func (e *Entry) Println(args ...any) {
	e.Logger.emit(InfoLevel, e.Data, sprintlnNoTrailingNewline(args...))
}
func (e *Entry) Printf(format string, args ...any) {
	e.Logger.emit(InfoLevel, e.Data, fmt.Sprintf(format, args...))
}
func (e *Entry) Debug(args ...any) { e.Logger.emit(DebugLevel, e.Data, fmt.Sprint(args...)) }
func (e *Entry) Info(args ...any)  { e.Logger.emit(InfoLevel, e.Data, fmt.Sprint(args...)) }
func (e *Entry) Infof(format string, args ...any) {
	e.Logger.emit(InfoLevel, e.Data, fmt.Sprintf(format, args...))
}

// Formatter renders an Entry into the bytes a Logger writes to its Out. A nil
// byte slice with a nil error suppresses the entry.
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// Logger is a small logrus-compatible facade backed by log/slog's Record and
// Handler types, so DDEV's existing output.UserOut/UserErr call sites
// (Printf-style methods, WithField chaining, direct Out/Level field access)
// didn't need to change when the backing implementation moved off logrus.
type Logger struct {
	Out       io.Writer
	Formatter Formatter
	Level     Level

	mu sync.Mutex
}

// New returns a Logger writing to os.Stderr at InfoLevel with a TextFormatter.
func New() *Logger {
	return &Logger{
		Out:       os.Stderr,
		Formatter: &TextFormatter{},
		Level:     InfoLevel,
	}
}

func (l *Logger) SetOutput(w io.Writer)    { l.Out = w }
func (l *Logger) SetFormatter(f Formatter) { l.Formatter = f }
func (l *Logger) SetLevel(level Level)     { l.Level = level }
func (l *Logger) GetLevel() Level          { return l.Level }

// Enabled implements slog.Handler.
func (l *Logger) Enabled(_ context.Context, level slog.Level) bool {
	return Level(level) >= l.Level
}

// Handle implements slog.Handler, rendering the record through Formatter and
// writing the result to Out.
func (l *Logger) Handle(_ context.Context, r slog.Record) error {
	data := make(Fields, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		data[a.Key] = a.Value.Any()
		return true
	})
	entry := &Entry{Logger: l, Time: r.Time, Level: Level(r.Level), Message: r.Message, Data: data}
	b, err := l.Formatter.Format(entry)
	if err != nil || b == nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.Out.Write(b)
	return err
}

// WithAttrs and WithGroup satisfy slog.Handler. DDEV attaches fields via
// WithField/WithFields instead of slog.Logger.With, so grouping is unused;
// both return the Logger unchanged.
func (l *Logger) WithAttrs(_ []slog.Attr) slog.Handler { return l }
func (l *Logger) WithGroup(_ string) slog.Handler      { return l }

// emit checks the level threshold, builds a slog.Record, and hands it to
// Handle, which slog.Handler implementations must not do on the caller's
// behalf.
func (l *Logger) emit(level Level, data Fields, msg string) {
	if !l.Enabled(context.Background(), slog.Level(level)) {
		return
	}
	r := slog.NewRecord(time.Now(), slog.Level(level), msg, 0)
	for k, v := range data {
		r.AddAttrs(slog.Any(k, v))
	}
	_ = l.Handle(context.Background(), r)
}

// sprintlnNoTrailingNewline joins args like fmt.Sprintln but without the
// trailing newline, since Formatter always appends one.
func sprintlnNoTrailingNewline(args ...any) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}

func (l *Logger) Print(args ...any)   { l.emit(InfoLevel, nil, fmt.Sprint(args...)) }
func (l *Logger) Println(args ...any) { l.emit(InfoLevel, nil, sprintlnNoTrailingNewline(args...)) }
func (l *Logger) Printf(format string, args ...any) {
	l.emit(InfoLevel, nil, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(args ...any)   { l.emit(DebugLevel, nil, fmt.Sprint(args...)) }
func (l *Logger) Debugln(args ...any) { l.emit(DebugLevel, nil, sprintlnNoTrailingNewline(args...)) }
func (l *Logger) Debugf(format string, args ...any) {
	l.emit(DebugLevel, nil, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(args ...any) { l.emit(InfoLevel, nil, fmt.Sprint(args...)) }
func (l *Logger) Infof(format string, args ...any) {
	l.emit(InfoLevel, nil, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(args ...any)    { l.emit(WarnLevel, nil, fmt.Sprint(args...)) }
func (l *Logger) Warning(args ...any) { l.Warn(args...) }
func (l *Logger) Warnln(args ...any)  { l.emit(WarnLevel, nil, sprintlnNoTrailingNewline(args...)) }
func (l *Logger) Warnf(format string, args ...any) {
	l.emit(WarnLevel, nil, fmt.Sprintf(format, args...))
}
func (l *Logger) Warningf(format string, args ...any) { l.Warnf(format, args...) }

func (l *Logger) Error(args ...any)   { l.emit(ErrorLevel, nil, fmt.Sprint(args...)) }
func (l *Logger) Errorln(args ...any) { l.emit(ErrorLevel, nil, sprintlnNoTrailingNewline(args...)) }
func (l *Logger) Errorf(format string, args ...any) {
	l.emit(ErrorLevel, nil, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(args ...any) { l.emit(FatalLevel, nil, fmt.Sprint(args...)); os.Exit(1) }
func (l *Logger) Fatalln(args ...any) {
	l.emit(FatalLevel, nil, sprintlnNoTrailingNewline(args...))
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, args ...any) {
	l.emit(FatalLevel, nil, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *Logger) Panic(args ...any) {
	msg := fmt.Sprint(args...)
	l.emit(PanicLevel, nil, msg)
	panic(msg)
}

// Exit terminates the process, matching logrus.Logger.Exit's signature.
func (l *Logger) Exit(code int) { os.Exit(code) }

// WithField returns an Entry carrying a single structured field, for chaining
// a message onto it: logger.WithField("key", v).Info("message").
func (l *Logger) WithField(key string, value any) *Entry {
	return &Entry{Logger: l, Data: Fields{key: value}}
}

// WithFields returns an Entry carrying several structured fields.
func (l *Logger) WithFields(fields Fields) *Entry {
	data := make(Fields, len(fields))
	maps.Copy(data, fields)
	return &Entry{Logger: l, Data: data}
}
