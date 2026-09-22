package applog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger writes redacted application logs to a file.
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
	w := io.Writer(f)
	// With -H windowsgui there is no console; never attach stdout (can spawn one).
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
	l.std.Printf("%s [%s] %s", time.Now().Format("01-02 15:04:05"), level, msg)
}

// Tail reads the last maxLines from the log file without loading huge heads fully into memory beyond a capped window.
func Tail(path string, maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 100
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	const maxWindow = 512 << 10 // 512 KiB
	size := st.Size()
	start := int64(0)
	if size > maxWindow {
		start = size - maxWindow
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	if start > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 && i+1 < len(data) {
			data = data[i+1:]
		}
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
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
