package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger writes redacted application logs to a file and stdout.
type Logger struct {
	mu   sync.Mutex
	file *os.File
	std  *log.Logger
}

func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	w := io.MultiWriter(os.Stdout, f)
	return &Logger{
		file: f,
		std:  log.New(w, "", 0),
	}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Info(format string, args ...any)  { l.write("INFO", format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.write("WARN", format, args...) }
func (l *Logger) Error(format string, args ...any) { l.write("ERROR", format, args...) }

func (l *Logger) write(level, format string, args ...any) {
	if l == nil || l.std == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = Redact(msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.std.Printf("%s [%s] %s", time.Now().Format(time.RFC3339), level, msg)
}

// Redact removes common secret patterns from log lines.
func Redact(s string) string {
	out := s
	for _, key := range []string{"password=", "uuid=", "secret=", "token="} {
		lower := strings.ToLower(out)
		idx := 0
		for {
			rel := strings.Index(lower[idx:], key)
			if rel < 0 {
				break
			}
			at := idx + rel
			start := at + len(key)
			end := start
			for end < len(out) {
				c := out[end]
				if c == ' ' || c == '"' || c == '\'' || c == ',' || c == ';' || c == '&' || c == '\n' || c == '\t' {
					break
				}
				end++
			}
			out = out[:start] + "***" + out[end:]
			lower = strings.ToLower(out)
			idx = start + 3
		}
	}
	for _, scheme := range []string{"vless://", "vmess://", "ss://"} {
		for {
			lower := strings.ToLower(out)
			i := strings.Index(lower, scheme)
			if i < 0 {
				break
			}
			j := i + len(scheme)
			for j < len(out) && out[j] != ' ' && out[j] != '"' && out[j] != '\n' && out[j] != '\t' {
				j++
			}
			out = out[:i] + "[redacted-link]" + out[j:]
		}
	}
	return out
}
