package plugin

import (
	"bufio"
	"encoding/json"
	"io"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// forwardLogs re-logs the JSON lines Serve's logger writes to stderr.
// Anything else (e.g. a panic trace) is logged as a warning line by line.
func forwardLogs(r io.Reader, logger *zap.Logger) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var entry map[string]any
		if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
			logger.Warn(sc.Text())
			continue
		}
		logEntry(logger, entry)
	}
}

func logEntry(logger *zap.Logger, entry map[string]any) {
	level := zapcore.InfoLevel
	if s, ok := entry["level"].(string); ok {
		if l, err := zapcore.ParseLevel(s); err == nil {
			// A module must not be able to panic or exit the host.
			level = min(l, zapcore.ErrorLevel)
		}
	}
	msg, _ := entry["msg"].(string)
	if name, ok := entry["logger"].(string); ok {
		logger = logger.Named(name)
	}
	for _, k := range []string{"level", "msg", "logger", "ts", "caller"} {
		delete(entry, k)
	}

	keys := make([]string, 0, len(entry))
	for k := range entry {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	fields := make([]zap.Field, 0, len(keys))
	for _, k := range keys {
		fields = append(fields, zap.Any(k, entry[k]))
	}
	if ce := logger.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}
