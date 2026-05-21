package mqtt

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
)

// zapHandler bridges slog to zap, so mochi-mqtt broker logs go to dolphin's log file.
type zapHandler struct {
	zap   *zap.SugaredLogger
	level slog.Leveler
	attrs []slog.Attr
	group string
}

func newZapHandler() *zapHandler {
	return &zapHandler{zap: zap.S(), level: slog.LevelDebug}
}

func (h *zapHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level.Level() }

func (h *zapHandler) Handle(_ context.Context, r slog.Record) error {
	fields := make([]any, 0, (r.NumAttrs()+len(h.attrs))*2)
	for _, a := range h.attrs {
		k, v := slogAttrToZap(a)
		fields = append(fields, k, v)
	}
	r.Attrs(func(a slog.Attr) bool {
		k, v := slogAttrToZap(a)
		fields = append(fields, k, v)
		return true
	})
	msg := r.Message
	if h.group != "" {
		msg = h.group + ": " + msg
	}
	switch r.Level {
	case slog.LevelDebug:
		h.zap.Debugw(msg, fields...)
	case slog.LevelInfo:
		h.zap.Infow(msg, fields...)
	case slog.LevelWarn:
		h.zap.Warnw(msg, fields...)
	case slog.LevelError:
		h.zap.Errorw(msg, fields...)
	}
	return nil
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	cp := *h
	cp.group = name
	return &cp
}

func slogAttrToZap(a slog.Attr) (any, any) {
	return a.Key, a.Value.Any()
}
