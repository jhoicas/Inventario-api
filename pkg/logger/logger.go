package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config opciones para el logger.
type Config struct {
	Env   string // development -> consola legible; production -> JSON
	Level string // trace, debug, info, warn, error
}

// Logger wrapper sobre zerolog para inyección y consistencia.
type Logger struct {
	zl zerolog.Logger
}

// New crea un logger estructurado. En development usa salida legible; en production JSON.
func New(cfg Config) *Logger {
	var w io.Writer = os.Stdout
	if cfg.Env == "development" {
		w = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	level := parseLevel(cfg.Level)
	zl := zerolog.New(w).Level(level).With().Timestamp().Logger()

	// Redirigir el logger global de zerolog para librerías que lo usen
	log.Logger = zl

	return &Logger{zl: zl}
}

func parseLevel(s string) zerolog.Level {
	switch s {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Trace, Debug, Info, Warn, Error delegados a zerolog.
func (l *Logger) Trace() *zerolog.Event { return l.zl.Trace() }
func (l *Logger) Debug() *zerolog.Event { return l.zl.Debug() }
func (l *Logger) Info() *zerolog.Event  { return l.zl.Info() }
func (l *Logger) Warn() *zerolog.Event  { return l.zl.Warn() }
func (l *Logger) Error() *zerolog.Event { return l.zl.Error() }
func (l *Logger) Fatal() *zerolog.Event { return l.zl.Fatal() }

// With crea un sublogger con campos fijos.
func (l *Logger) With() zerolog.Context {
	return l.zl.With()
}

// Zerolog devuelve el logger interno por si se necesita la API directa.
func (l *Logger) Zerolog() zerolog.Logger {
	return l.zl
}
