package ui

import (
	"fmt"
	"os"

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

func Progress(source, project, detail string) {
	msg := fmt.Sprintf("Fetching %s for %s", source, project)
	if detail != "" {
		msg += " " + color(dim, "("+detail+")")
	} else if colorEnabled {
		msg = color(cyan, msg)
	}
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func Done(source, project string, count int) {
	fmt.Fprintf(os.Stderr, "%s\n", color(green, fmt.Sprintf("  ✓ %s/%s: %d records", source, project, count)))
}

func Skip(source, project, reason string) {
	fmt.Fprintf(os.Stderr, "%s\n", color(dim, fmt.Sprintf("  · %s/%s: %s", source, project, reason)))
}
