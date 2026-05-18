package printer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/lczyk/trace"
)

// chromeEvent is one entry in the Chrome Trace Event Format (catapult)
// JSON array consumed by chrome://tracing, perfetto, speedscope, and
// the like. Spec: https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU
type chromeEvent struct {
	Name string         `json:"name"`
	Ph   string         `json:"ph"`             // phase: "B" begin, "E" end, "i" instant
	Ts   int64          `json:"ts"`             // microseconds from trace start
	Pid  int            `json:"pid"`            // process id (hardcoded 1)
	Tid  int            `json:"tid"`            // thread id (hardcoded 1; future: per-goroutine)
	Args map[string]any `json:"args,omitempty"` // instant event payload
}

// ErrNotTimed is returned by [PrintChromeTrace] when the supplied
// tracer was not constructed with [trace.NewTracerWithTime] -- Chrome
// trace output requires per-node timestamps to be meaningful.
var ErrNotTimed = errors.New("chrome trace requires a tracer constructed with NewTracerWithTime")

// PrintChromeTrace writes tr's recorded trace to w as a Chrome Trace
// Event Format JSON array. The tracer must have been constructed with
// [trace.NewTracerWithTime]; otherwise [ErrNotTimed] is returned.
//
// Enter / Exit nodes become paired "B" / "E" events. Message nodes
// become "i" (instant) events with the message body in args.text. All
// events use pid=1, tid=1 -- single-tracer view for now.
func PrintChromeTrace(tr trace.Tracer, w io.Writer) error {
	walkable, err := tr.ToWalkable()
	if err != nil {
		return err
	}

	var events []chromeEvent
	var start time.Time

	err = walkable.Walk(func(n trace.Node) error {
		var (
			name string
			ph   string
			ts   time.Time
			args map[string]any
		)
		switch n := n.(type) {
		case *trace.Enter:
			name, ph, ts = n.Name(), "B", n.Time()
		case *trace.Exit:
			name, ph, ts = n.Name(), "E", n.Time()
		case *trace.Message:
			name, ph, ts = n.Message, "i", n.Time()
			if scope := n.ParentName(); scope != "" {
				args = map[string]any{"scope": scope}
			}
		default:
			return nil
		}
		if ts.IsZero() {
			return ErrNotTimed
		}
		if start.IsZero() {
			start = ts
		}
		events = append(events, chromeEvent{
			Name: name,
			Ph:   ph,
			Ts:   ts.Sub(start).Microseconds(),
			Pid:  1,
			Tid:  1,
			Args: args,
		})
		return nil
	})
	if err != nil {
		// Walk wraps callback errors; unwrap to surface ErrNotTimed cleanly
		if errors.Is(err, ErrNotTimed) {
			return ErrNotTimed
		}
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(events)
}

// PrintChromeTraceCtx is the context-carried convenience over
// [PrintChromeTrace]. No-op when ctx carries no tracer.
func PrintChromeTraceCtx(ctx context.Context, w io.Writer) error {
	tracer := trace.GetTracer(ctx)
	if tracer == nil {
		return nil
	}
	return PrintChromeTrace(tracer, w)
}
