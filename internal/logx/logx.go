package logx

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Field represents a structured logging field.
type Field struct {
	Key   string
	Value interface{}
}

// Info logs at info level.
func Info(component, op, msg string, fields ...Field) {
	emitLog("info", component, op, msg, nil, fields)
}

// Warn logs at warn level.
func Warn(component, op, msg string, fields ...Field) {
	emitLog("warn", component, op, msg, nil, fields)
}

// Error logs at error level and records an error value.
func Error(component, op, msg string, err error, fields ...Field) {
	emitLog("error", component, op, msg, err, fields)
}

func emitLog(level, component, op, msg string, err error, fields []Field) {
	var sb strings.Builder
	sb.WriteString("ts=")
	sb.WriteString(time.Now().UTC().Format(time.RFC3339Nano))
	sb.WriteString(" level=")
	sb.WriteString(level)
	sb.WriteString(" component=")
	sb.WriteString(formatValue(component))
	sb.WriteString(" op=")
	sb.WriteString(formatValue(op))
	sb.WriteString(" msg=")
	sb.WriteString(formatValue(msg))
	if err != nil {
		sb.WriteString(" err=")
		sb.WriteString(formatValue(err))
	}
	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		sb.WriteByte(' ')
		sb.WriteString(formatField(field))
	}
	fmt.Fprintln(os.Stderr, sb.String())
}

func formatField(field Field) string {
	return field.Key + "=" + formatValue(field.Value)
}

func formatValue(value interface{}) string {
	s := fmt.Sprint(value)
	var b strings.Builder
	needsQuote := false
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString("\\n")
			needsQuote = true
		case '\r':
			b.WriteString("\\r")
			needsQuote = true
		case '\t':
			b.WriteString("\\t")
			needsQuote = true
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if r < 0x20 || r > 0x7E {
				fmt.Fprintf(&b, "\\u%04x", r)
				needsQuote = true
			} else {
				b.WriteRune(r)
				if r == ' ' {
					needsQuote = true
				}
			}
		}
	}
	valueStr := b.String()
	if needsQuote {
		return "\"" + valueStr + "\""
	}
	return valueStr
}
