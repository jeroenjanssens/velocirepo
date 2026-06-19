package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-isatty"
)

var colorEnabled = isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

const (
	reset  = "\033[0m"
	dim    = "\033[2m"
	red    = "\033[31m"
	yellow = "\033[33m"
	green  = "\033[32m"
	cyan   = "\033[36m"
)

func color(c, s string) string {
	if !colorEnabled {
		return s
	}
	return c + s + reset
}

func prefix(source, project string) string {
	return fmt.Sprintf("[%s/%s]", source, project)
}

func Info(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(dim, msg))
}

func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

func Success(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(green, msg))
}

func Successf(format string, args ...interface{}) {
	Success(fmt.Sprintf(format, args...))
}

func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(yellow, "Warning: "+msg))
}

func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(red, "Error: "+msg))
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

func FetchStart(source, project, dateRange string) {
	msg := fmt.Sprintf("%s fetching %s", prefix(source, project), dateRange)
	fmt.Fprintf(os.Stderr, "%s\n", color(cyan, msg))
}

func FetchDone(source, project string, count int, duration time.Duration) {
	msg := fmt.Sprintf("%s %d records in %s", prefix(source, project), count, formatDuration(duration))
	fmt.Fprintf(os.Stderr, "%s\n", color(green, "  ✓ "+msg))
}

func FetchSkip(source, project, reason string) {
	msg := fmt.Sprintf("%s skipped: %s", prefix(source, project), reason)
	fmt.Fprintf(os.Stderr, "%s\n", color(dim, "  · "+msg))
}

func FetchError(source, project string, err error) {
	msg := fmt.Sprintf("%s %v", prefix(source, project), err)
	fmt.Fprintf(os.Stderr, "%s\n", color(red, "  ✗ "+msg))
}

func FetchWarn(source, project, msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(yellow, fmt.Sprintf("  ! %s %s", prefix(source, project), msg)))
}


func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
