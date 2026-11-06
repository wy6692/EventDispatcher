package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	timeLayout = "2006-01-02 15:04:05.000"
	levelWidth = 5
	// 现有日志文案最长约 44 个字符，固定到 48，字段从同一列开始。
	msgWidth = 48
)

type consoleHandler struct {
	w      io.Writer
	level  slog.Level
	color  bool
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func newConsoleHandler(w io.Writer, level slog.Level, color bool) *consoleHandler {
	return &consoleHandler{
		w:     w,
		level: level,
		color: color,
		mu:    &sync.Mutex{},
	}
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	var line bytes.Buffer
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	h.writePainted(&line, "\033[90m", ts.Local().Format(timeLayout))
	h.writeSep(&line)
	h.writePainted(&line, levelColor(r.Level), padRight(levelLabel(r.Level), levelWidth))
	h.writeSep(&line)

	msg := strings.ReplaceAll(r.Message, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", "")
	line.WriteString(padRight(msg, msgWidth))

	var attrs bytes.Buffer
	writeAttrs(&attrs, h.groups, h.attrs, h.color)
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&attrs, h.groups, a, h.color)
		return true
	})
	if attrs.Len() > 0 {
		h.writeSep(&line)
		line.Write(attrs.Bytes())
	}
	line.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(line.Bytes())
	return err
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.groups = append(append([]string{}, h.groups...), name)
	return &next
}

func (h *consoleHandler) writeSep(b *bytes.Buffer) {
	h.writePainted(b, "\033[90m", " | ")
}

func (h *consoleHandler) writePainted(b *bytes.Buffer, color, text string) {
	if h.color {
		b.WriteString(color)
	}
	b.WriteString(text)
	if h.color {
		b.WriteString("\033[0m")
	}
}

func levelLabel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "\033[1;31m"
	case level >= slog.LevelWarn:
		return "\033[33m"
	case level >= slog.LevelInfo:
		return "\033[32m"
	default:
		return "\033[36m"
	}
}

func writeAttrs(b *bytes.Buffer, groups []string, attrs []slog.Attr, color bool) {
	for _, attr := range attrs {
		writeAttr(b, groups, attr, color)
	}
}

func writeAttr(b *bytes.Buffer, groups []string, attr slog.Attr, color bool) {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		next := groups
		if attr.Key != "" {
			next = append(append([]string{}, groups...), attr.Key)
		}
		writeAttrs(b, next, attr.Value.Group(), color)
		return
	}
	if attr.Key == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("  ")
	}
	key := attr.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	if color {
		b.WriteString("\033[36m")
		b.WriteString(key)
		b.WriteString("\033[0m=")
	} else {
		b.WriteString(key)
		b.WriteByte('=')
	}
	b.WriteString(formatValue(attr.Value))
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return quoteIfNeeded(v.String())
	case slog.KindTime:
		return v.Time().Local().Format(timeLayout)
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			return quoteIfNeeded(err.Error())
		}
		return quoteIfNeeded(fmt.Sprint(v.Any()))
	default:
		return v.String()
	}
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	for _, r := range s {
		if r <= ' ' || r == '"' || r == '=' || r == '\\' || unicode.IsControl(r) {
			return strconv.Quote(s)
		}
	}
	return s
}

func padRight(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		switch {
		case r <= 0x1F || (r >= 0x7F && r <= 0x9F):
			continue
		case isWideRune(r):
			n += 2
		default:
			n++
		}
	}
	return n
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x2E80 && r <= 0xA4CF) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1F300 && r <= 0x1F64F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x20000 && r <= 0x3FFFD)
}
