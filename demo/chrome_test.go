//go:build demo_chrome

// Separate build tag so the chrome demo doesn't run with `make demo`
// (it produces a file artefact rather than terminal output). Run with:
//
//	go test -tags demo_chrome -run TestDemoChromeTrace ./demo/
//
// or `make demo-chrome`.
package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

// TestDemoChromeTrace produces a Chrome Trace Event Format JSON file
// alongside this test. Load it in chrome://tracing or perfetto.dev/ui
// to see the call tree as an interactive flame graph.
func TestDemoChromeTrace(t *testing.T) {
	tr := trace.NewTracerWithTime()
	parse(tr)

	// Resolve path relative to this test's working dir (demo/).
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cwd, "trace.chrome.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := printer.PrintChromeTrace(tr, f); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("wrote %s -- open in chrome://tracing or perfetto.dev/ui\n", path)
}

// parse simulates a small parser doing multi-level work with delays so
// the chrome timeline has visible duration.
func parse(tr trace.Tracer) {
	defer tr.Un(tr.Trace("parse"))
	tr.Message("starting")
	time.Sleep(2 * time.Millisecond)
	for i := 0; i < 3; i++ {
		statement(tr, i)
	}
	tr.Message("done")
}

func statement(tr trace.Tracer, i int) {
	defer tr.Un(tr.Trace("statement"))
	tr.Messagef("stmt %d", i)
	time.Sleep(1 * time.Millisecond)
	expression(tr)
}

func expression(tr trace.Tracer) {
	defer tr.Un(tr.Trace("expression"))
	lex(tr)
	time.Sleep(500 * time.Microsecond)
}

func lex(tr trace.Tracer) {
	defer tr.Un(tr.Trace("lex"))
	for _, tok := range []string{"INT", "PLUS", "INT"} {
		tr.Messagef("token %s", tok)
	}
	time.Sleep(300 * time.Microsecond)
}
