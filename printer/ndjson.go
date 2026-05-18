package printer

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/lczyk/trace"
)

// ndjsonNode is the wire shape for one node. Fields are omitted when
// empty so consumers (jq, etc.) don't see noise from absent values.
type ndjsonNode struct {
	Kind    string `json:"kind"`              // "enter" / "exit" / "message"
	Name    string `json:"name,omitempty"`    // enter/exit scope name
	Scope   string `json:"scope,omitempty"`   // message: enclosing scope name
	Message string `json:"message,omitempty"` // message body
	T       string `json:"t,omitempty"`       // RFC3339Nano, only when tracer was timed
}

// PrintNDJSON writes one JSON object per line per node from tr to w.
// Time fields are present only when the tracer was constructed with
// [trace.NewTracerWithTime].
//
// The output is newline-delimited JSON (NDJSON): pipe directly into
// `jq -c` or any line-oriented JSON consumer.
func PrintNDJSON(tr trace.Tracer, w io.Writer) error {
	walkable, err := tr.ToWalkable()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	return walkable.Walk(func(n trace.Node) error {
		var obj ndjsonNode
		switch n := n.(type) {
		case *trace.Enter:
			obj.Kind = "enter"
			obj.Name = n.Name()
			obj.T = formatTime(n.Time())
		case *trace.Exit:
			obj.Kind = "exit"
			obj.Name = n.Name()
			obj.T = formatTime(n.Time())
		case *trace.Message:
			obj.Kind = "message"
			obj.Scope = n.ParentName()
			obj.Message = n.Message
			obj.T = formatTime(n.Time())
		default:
			return nil
		}
		return enc.Encode(obj)
	})
}

// PrintNDJSONCtx is the context-carried convenience over [PrintNDJSON].
// No-op when ctx carries no tracer.
func PrintNDJSONCtx(ctx context.Context, w io.Writer) error {
	tracer := trace.GetTracer(ctx)
	if tracer == nil {
		return nil
	}
	return PrintNDJSON(tracer, w)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
