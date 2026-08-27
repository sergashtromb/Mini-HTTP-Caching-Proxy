//this file is needed for define a log system

package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type LogHandler struct {
	level 	slog.Level
	out 	io.Writer
}

func New(out io.Writer, level slog.Level) *LogHandler {
	return &LogHandler{
		out: out,
		level: level,
	}
} 

func (lh *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= lh.level.Level()
}

func (lh *LogHandler) Handle(ctx context.Context, r slog.Record) error {

	log_string := make([]byte, 0, 1024)

	log_string = append(log_string, byte('['))
	log_string = append(log_string, []byte(levelToString(r.Level))...)
	log_string = append(log_string, byte(']'))
	log_string = append(log_string, byte(' '))

	if !r.Time.IsZero() {
		log_string = append(log_string, []byte(r.Time.Format("02-01-2006 15:04"))...)
		log_string = append(log_string, byte(' '))
	}

	log_string = append(log_string, []byte(r.Message)...)
	log_string = append(log_string, byte(' '))

	r.Attrs(func(a slog.Attr) bool {
		log_string = fmt.Appendf(log_string, " %s=%v ", a.Key, a.Value)
		return true
	})

	log_string = append(log_string, '\n')

	_, err := lh.out.Write(log_string)
	if err != nil {
		fmt.Println("Error write log err: ", err)
		return err
	}

	return nil
}

func (lh *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// don't support attrs
	return lh
}

func (lh *LogHandler) WithGroup(name string) slog.Handler {
	// don't support group
	return lh
}

func levelToString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelError:
		return "ERROR"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelInfo:
		return "INFO"
	default:
		return "????"
	}
}

func stringToLevel(level string) slog.Level {

	lvlstring := strings.ToLower(strings.TrimSpace(level))

	switch lvlstring {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}

}

func Init(path, lvlstring string) (*os.File, error) {

	err := os.Mkdir(path, 0755)
	if err != nil {}

	file, err := os.OpenFile(filepath.Join(path, "proxy.log"), os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Error create log file err:", err)
		return  nil, err
	}

	level := stringToLevel(lvlstring)
	out := io.MultiWriter(os.Stdout, file)

	logHandler := New(out, level)
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	return file, nil
}