//go:build demo

package demo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

// helper to print trace output for the demo
func dump(t *testing.T, tr trace.Tracer, messages bool) {
	t.Helper()
	printer.PrintTrace(tr, messages, os.Stdout)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoBasic(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("outer"))
		func() {
			defer tr.Un(tr.Trace("middle"))
			func() {
				defer tr.Un(tr.Trace("inner"))
			}()
		}()
	}()
	dump(t, tr, false)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoMessages(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("parse"))
		tr.Message("starting")
		func() {
			defer tr.Un(tr.Trace("lex"))
			tr.Messagef("token %s", "INT")
			tr.Messagef("token %s", "PLUS")
		}()
		tr.Message("done")
	}()
	dump(t, tr, true)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoGap(t *testing.T) {
	// intermediate function not registered with the tracer
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("first"))
		tr.Message("from first")
		// no Trace here -- gap
		func() {
			tr.Message("from untraced second")
			func() {
				defer tr.Un(tr.Trace("third"))
				tr.Message("from third")
			}()
		}()
	}()
	dump(t, tr, true)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoDisabled(t *testing.T) {
	// SetMessagesEnabled gates Message/Messagef at runtime
	tr := trace.NewTracer()
	tr.SetMessagesEnabled(false)
	func() {
		defer tr.Un(tr.Trace("scope"))
		tr.Message("dropped")
		tr.SetMessagesEnabled(true)
		tr.Message("kept")
	}()
	dump(t, tr, true)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoMultiline(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("a"))
		func() {
			defer tr.Un(tr.Trace("b"))
			func() {
				defer tr.Un(tr.Trace("c"))
				tr.Message("deeply nested message")
			}()
		}()
	}()
	for _, m := range tr.Messages() {
		printer.PrettyPrintMessage(m, printer.MULTILINE)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoNDJSON(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("parse"))
		tr.Message("starting")
		func() {
			defer tr.Un(tr.Trace("lex"))
			tr.Messagef("token %s", "INT")
		}()
	}()
	if err := printer.PrintNDJSON(tr, os.Stdout); err != nil {
		t.Fatal(err)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoReset(t *testing.T) {
	tr := trace.NewTracer()
	for i := 0; i < 3; i++ {
		tr.Reset()
		func() {
			defer tr.Un(tr.Trace("cycle"))
			tr.Messagef("iteration %d", i)
		}()
		dump(t, tr, true)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoTimed(t *testing.T) {
	tr := trace.NewTracerWithTime()
	defer tr.Un(tr.Trace("work"))
	tr.Message("step")
	w, err := tr.ToWalkable()
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Walk(func(n trace.Node) error {
		switch n := n.(type) {
		case *trace.Enter:
			if n.Name() == "<START>" {
				return nil
			}
			fmt.Fprintf(os.Stdout, "ENTER %s at %s\n", n.Name(), n.Time().Format("15:04:05.000000"))
		case *trace.Exit:
			if n.Name() == "<END>" {
				return nil
			}
			fmt.Fprintf(os.Stdout, "EXIT  %s at %s\n", n.Name(), n.Time().Format("15:04:05.000000"))
		case *trace.Message:
			fmt.Fprintf(os.Stdout, "MSG   [%s] %s at %s\n", n.ParentName(), n.Message, n.Time().Format("15:04:05.000000"))
		}
		return nil
	})
}

////////////////////////////////////////////////////////////////////////////////

// TestDemoChromeTrace writes a Chrome Trace Event Format JSON file
// alongside this test (demo/trace.chrome.json, gitignored). Load it in
// chrome://tracing or ui.perfetto.dev to see the call tree as an
// interactive flame graph.
func TestDemoChromeTrace(t *testing.T) {
	tr := trace.NewTracerWithTime()
	parseDemo(tr)

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
	fmt.Printf("wrote %s -- open in chrome://tracing or ui.perfetto.dev\n", path)
}

func parseDemo(tr trace.Tracer) {
	defer tr.Un(tr.Trace("parse"))
	tr.Message("starting")
	time.Sleep(2 * time.Millisecond)
	for i := 0; i < 3; i++ {
		statementDemo(tr, i)
	}
	tr.Message("done")
}

func statementDemo(tr trace.Tracer, i int) {
	defer tr.Un(tr.Trace("statement"))
	tr.Messagef("stmt %d", i)
	time.Sleep(1 * time.Millisecond)
	expressionDemo(tr)
}

func expressionDemo(tr trace.Tracer) {
	defer tr.Un(tr.Trace("expression"))
	lexDemo(tr)
	time.Sleep(500 * time.Microsecond)
}

func lexDemo(tr trace.Tracer) {
	defer tr.Un(tr.Trace("lex"))
	for _, tok := range []string{"INT", "PLUS", "INT"} {
		tr.Messagef("token %s", tok)
	}
	time.Sleep(300 * time.Microsecond)
}

////////////////////////////////////////////////////////////////////////////////

func TestDemoContext(t *testing.T) {
	tr := trace.NewTracer()
	ctx := trace.WithTracer(context.Background(), tr)
	work(ctx)
	dump(t, tr, true)
}

func work(ctx context.Context) {
	defer trace.TraceCtx(ctx)()
	trace.MessageStrCtx(ctx, "fast path")
	helper(ctx)
}

func helper(ctx context.Context) {
	defer trace.TraceCtx(ctx)()
	if trace.IsTracing(ctx) {
		trace.MessagefCtx(ctx, "guarded %d", 42)
	}
	trace.MessageCtx(ctx, "anything", " ", fmt.Sprintf("%d", 7))
}
