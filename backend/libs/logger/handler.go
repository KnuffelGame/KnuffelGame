package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// color codes
const (
	colorReset  = "\x1b[0m"
	colorBlue   = "\x1b[34m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorRed    = "\x1b[31m"
)

func colorForLevel(l slog.Level) string {
	if l <= slog.LevelDebug {
		return colorBlue
	}
	if l == slog.LevelInfo {
		return colorGreen
	}
	if l == slog.LevelWarn {
		return colorYellow
	}
	return colorRed
}

// colorJSONHandler is a custom slog.Handler that outputs structured JSON per record and optionally wraps the full line in ANSI color codes based on level.
// Concurrency: writes guarded by mutex to keep each line atomic.
// NOTE: ANSI color wrapping renders the raw line non-strict JSON. Disable Color for strict ingestion.

type colorJSONHandler struct {
	w         io.Writer
	minLevel  slog.Level
	addSource bool
	service   string
	color     bool
	mu        sync.Mutex
	groups    []string    // nested group names
	static    []slog.Attr // attrs added via WithAttrs
}

func newColorJSONHandler(w io.Writer, opts Options) slog.Handler {
	return &colorJSONHandler{
		w:         w,
		minLevel:  opts.Level,
		addSource: opts.AddSource,
		service:   opts.Service,
		color:     opts.Color,
	}
}

func (h *colorJSONHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.minLevel }

func (h *colorJSONHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(nil, r.Level) {
		return nil
	}
	data := make(map[string]interface{}, 8+len(h.static))
	data["time"] = r.Time.UTC().Format(time.RFC3339Nano)
	data["level"] = r.Level.String()
	data["msg"] = r.Message
	if h.service != "" {
		data["service"] = h.service
	}
	if h.addSource {
		if frame, ok := frameForPC(r.PC); ok {
			data["source"] = frame.File + ":" + itoa(frame.Line)
		}
	}
	for _, a := range h.static {
		addAttr(data, h.groups, a)
	}
	r.Attrs(func(a slog.Attr) bool { addAttr(data, h.groups, a); return true })
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	line := b
	if h.color {
		line = []byte(colorForLevel(r.Level) + string(b) + colorReset)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.w.Write(append(line, '\n'))
	return err
}

func (h *colorJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newStatic := append([]slog.Attr{}, h.static...)
	newStatic = append(newStatic, attrs...)
	return &colorJSONHandler{
		w:         h.w,
		minLevel:  h.minLevel,
		addSource: h.addSource,
		service:   h.service,
		color:     h.color,
		groups:    append([]string{}, h.groups...),
		static:    newStatic,
	}
}

func (h *colorJSONHandler) WithGroup(name string) slog.Handler {
	return &colorJSONHandler{
		w:         h.w,
		minLevel:  h.minLevel,
		addSource: h.addSource,
		service:   h.service,
		color:     h.color,
		groups:    append(append([]string{}, h.groups...), name),
		static:    append([]slog.Attr{}, h.static...),
	}
}

func addAttr(dst map[string]interface{}, groups []string, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	v := attrValueToAny(a.Value)
	if len(groups) == 0 {
		dst[a.Key] = v
		return
	}
	m := dst
	for _, g := range groups {
		if existing, ok := m[g]; ok {
			if em, ok2 := existing.(map[string]interface{}); ok2 {
				m = em
				continue
			}
		}
		newMap := make(map[string]interface{})
		m[g] = newMap
		m = newMap
	}
	m[a.Key] = v
}

func attrValueToAny(v slog.Value) interface{} {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		out := make(map[string]interface{})
		for _, a := range v.Group() {
			addAttr(out, nil, a)
		}
		return out
	case slog.KindAny:
		return v.Any()
	default:
		return v.String()
	}
}

func frameForPC(pc uintptr) (runtime.Frame, bool) {
	if pc == 0 {
		return runtime.Frame{}, false
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	f, _ := frames.Next()
	return f, true
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	bp := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		bp--
		b[bp] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		bp--
		b[bp] = '-'
	}
	return string(b[bp:])
}
