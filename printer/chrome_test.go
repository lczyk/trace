package printer_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

func TestPrintChromeTrace(t *testing.T) {
	tr := trace.NewTracerWithTime()
	func() {
		defer tr.Un(tr.Trace("outer"))
		tr.Message("hi")
		func() {
			defer tr.Un(tr.Trace("inner"))
		}()
	}()
	var buf bytes.Buffer
	if err := printer.PrintChromeTrace(tr, &buf); err != nil {
		t.Fatalf("PrintChromeTrace: %v", err)
	}
	var events []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &events); err != nil {
		t.Fatalf("invalid JSON: %v -- %s", err, buf.String())
	}
	// expect: B(START), B(outer), i(hi), B(inner), E(inner), E(outer), E(END)
	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}
	wantPhases := []string{"B", "B", "i", "B", "E", "E", "E"}
	for i, e := range events {
		if e["ph"] != wantPhases[i] {
			t.Fatalf("event %d: ph=%v want %s", i, e["ph"], wantPhases[i])
		}
		if _, ok := e["ts"]; !ok {
			t.Fatalf("event %d: missing ts", i)
		}
	}
	// instant event carries scope in args
	msg := events[2]
	args, ok := msg["args"].(map[string]any)
	if !ok || args["scope"] != "outer" {
		t.Fatalf("expected instant event with args.scope=outer, got %v", msg)
	}
}

func TestPrintChromeTraceUntimedErrors(t *testing.T) {
	tr := trace.NewTracer() // not timed
	tr.Un(tr.Trace("x"))
	var buf bytes.Buffer
	err := printer.PrintChromeTrace(tr, &buf)
	if !errors.Is(err, printer.ErrNotTimed) {
		t.Fatalf("expected ErrNotTimed, got %v", err)
	}
}
