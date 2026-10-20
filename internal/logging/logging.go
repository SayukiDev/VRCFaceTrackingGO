// Package logging builds the application zap logger: a file sink, stdout
// and an in-memory buffer that feeds the Output page.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Line is one rendered log line for the UI.
type Line struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Buffer keeps every log line and notifies a listener of new ones.
type Buffer struct {
	mu     sync.Mutex
	lines  []Line
	onLine func(Line)
}

const maxBufferedLines = 5000

// SetListener registers a callback invoked for every new line.
func (b *Buffer) SetListener(fn func(Line)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onLine = fn
}

// Lines returns a copy of every buffered line.
func (b *Buffer) Lines() []Line {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Line, len(b.lines))
	copy(out, b.lines)
	return out
}

// Text renders the whole buffer as plain text.
func (b *Buffer) Text() string {
	var sb strings.Builder
	for _, l := range b.Lines() {
		sb.WriteString(l.Message)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (b *Buffer) add(l Line) {
	b.mu.Lock()
	b.lines = append(b.lines, l)
	if len(b.lines) > maxBufferedLines {
		b.lines = b.lines[len(b.lines)-maxBufferedLines:]
	}
	fn := b.onLine
	b.mu.Unlock()
	if fn != nil {
		fn(l)
	}
}

// bufferCore is a zapcore.Core that renders entries into Buffer in the
// "[Category] Level: message" style of the original application.
type bufferCore struct {
	zapcore.LevelEnabler
	buf    *Buffer
	enc    zapcore.Encoder
	fields []zapcore.Field
}

func newBufferCore(buf *Buffer, level zapcore.LevelEnabler) *bufferCore {
	cfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LineEnding:     "",
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	return &bufferCore{LevelEnabler: level, buf: buf, enc: zapcore.NewConsoleEncoder(cfg)}
}

func (c *bufferCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(append([]zapcore.Field{}, c.fields...), fields...)
	return &clone
}

func (c *bufferCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *bufferCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	all := append(append([]zapcore.Field{}, c.fields...), fields...)
	rendered, err := c.enc.EncodeEntry(e, all)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(rendered.String())
	rendered.Free()

	name := e.LoggerName
	if name == "" {
		name = "App"
	}
	c.buf.add(Line{
		Level:   e.Level.CapitalString(),
		Message: fmt.Sprintf("[%s] %s: %s", name, e.Level.CapitalString(), text),
	})
	return nil
}

func (c *bufferCore) Sync() error { return nil }

// New creates the application logger. logDir receives latest.log.
func New(logDir string, debug bool) (*zap.Logger, *Buffer, error) {
	buf := &Buffer{}
	cores := []zapcore.Core{newBufferCore(buf, zapcore.DebugLevel)}

	fileCfg := zap.NewDevelopmentEncoderConfig()
	fileCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEnc := zapcore.NewConsoleEncoder(fileCfg)

	if err := os.MkdirAll(logDir, 0o755); err == nil {
		f, err := os.Create(filepath.Join(logDir, "latest.log"))
		if err == nil {
			cores = append(cores, zapcore.NewCore(fileEnc, zapcore.Lock(f), zapcore.DebugLevel))
		}
	}
	if debug {
		cores = append(cores, zapcore.NewCore(fileEnc, zapcore.Lock(os.Stdout), zapcore.DebugLevel))
	}

	logger := zap.New(zapcore.NewTee(cores...))
	return logger, buf, nil
}
