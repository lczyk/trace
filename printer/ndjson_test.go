package printer_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

func TestPrintNDJSON(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("outer"))
		tr.Message("hi")
		func() {
			defer tr.Un(tr.Trace("inner"))
		}()
	}()
	var buf bytes.Buffer
	if err := printer.PrintNDJSON(tr, &buf); err != nil {
		t.Fatalf("PrintNDJSON: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// expect: enter(START), enter(outer), message(hi), enter(inner), exit(inner), exit(outer), exit(END)
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d: %s", len(lines), buf.String())
	}
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("line %d not valid JSON: %v -- %s", i, err, line)
		}
		if _, ok := obj["kind"]; !ok {
			t.Fatalf("line %d missing kind: %s", i, line)
		}
	}
}

func TestPrintNDJSONTimed(t *testing.T) {
	tr := trace.NewTracerWithTime()
	tr.Un(tr.Trace("x"))
	var buf bytes.Buffer
	if err := printer.PrintNDJSON(tr, &buf); err != nil {
		t.Fatalf("PrintNDJSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"t":"`) {
		t.Fatalf("expected timestamps in output: %s", buf.String())
	}
}

func TestPrintNDJSONUntimedOmitsT(t *testing.T) {
	tr := trace.NewTracer()
	tr.Un(tr.Trace("x"))
	var buf bytes.Buffer
	_ = printer.PrintNDJSON(tr, &buf)
	if strings.Contains(buf.String(), `"t":`) {
		t.Fatalf("expected no t field for untimed tracer: %s", buf.String())
	}
}
