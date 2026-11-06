package logger

import (
	"bytes"
	"context"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestConsoleColumnsAlign(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelDebug, false)
	log.Info("EventDispatcher server started", slog.String("addr", ":8080"))
	log.Error("failed to load configuration", slog.String("error", "必须设置环境变量 ADMIN_TOKEN"))
	log.Warn("delivery attempt failed", slog.String("deliveryId", "d1"), slog.Int("attempt", 2))
	log.Info("启动完成", slog.String("addr", ":9090"))

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines=%d\n%s", len(lines), buf.String())
	}
	cols := []int{
		columnOf(lines[0], "addr="),
		columnOf(lines[1], "error="),
		columnOf(lines[2], "deliveryId="),
		columnOf(lines[3], "addr="),
	}
	for i := 1; i < len(cols); i++ {
		if cols[i] < 0 || cols[i] != cols[0] {
			t.Fatalf("columns %v\n%s", cols, buf.String())
		}
	}
	infoAt := strings.Index(lines[0], "INFO")
	if infoAt != strings.Index(lines[1], "ERROR") || infoAt != strings.Index(lines[2], "WARN") {
		t.Fatalf("level column drifted\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "\033") {
		t.Fatal("buffer output should not include color")
	}
}

func TestConsoleColorKeepsColumns(t *testing.T) {
	var buf bytes.Buffer
	h := newConsoleHandler(&buf, slog.LevelDebug, true)
	log := slog.New(&Handler{Handler: h})
	log.Info("EventDispatcher server started", slog.String("addr", ":8080"))
	log.Error("failed to load configuration", slog.String("error", "token required"))

	plain := stripANSI(buf.String())
	lines := strings.Split(strings.TrimRight(plain, "\n"), "\n")
	if strings.Index(lines[0], "addr=") != strings.Index(lines[1], "error=") {
		t.Fatalf("colored columns drifted\n%s", plain)
	}
	if !strings.Contains(buf.String(), "\033[32m") || !strings.Contains(buf.String(), "\033[1;31m") {
		t.Fatalf("missing level color\n%s", buf.String())
	}
}

func TestConsoleIncludesTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, false).With(slog.String("component", "http"))
	log.InfoContext(WithTraceID(context.Background(), "abc123"), "ingest event processed", slog.String("event", "subscribe"))

	text := buf.String()
	for _, want := range []string{"component=http", "event=subscribe", "traceId=abc123"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s\n%s", want, text)
		}
	}
	fields := strings.Index(text, "component=")
	if fields < 0 || strings.Index(text, "event=") < fields || strings.Index(text, "traceId=") < fields {
		t.Fatalf("field order\n%s", text)
	}
}

func TestConsoleDropsDebugAndQuotesSpaces(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, false)
	log.Debug("hidden detail", slog.String("k", "v"))
	log.Info("shown", slog.String("error", "boom now"))
	text := buf.String()
	if strings.Contains(text, "hidden detail") {
		t.Fatal(text)
	}
	if !strings.Contains(text, `error="boom now"`) {
		t.Fatal(text)
	}
}

func TestJSONFormatUnchanged(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, slog.LevelInfo, true)
	log.Info("hello", slog.String("addr", ":8080"))
	if !strings.Contains(buf.String(), `"msg":"hello"`) || !strings.Contains(buf.String(), `"addr":":8080"`) {
		t.Fatal(buf.String())
	}
}

func columnOf(line, token string) int {
	i := strings.Index(line, token)
	if i < 0 {
		return -1
	}
	return displayWidth(line[:i])
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
