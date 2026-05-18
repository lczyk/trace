package trace_test

import (
	"context"
	"os"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

func ExampleNewTracer() {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("outer"))
		func() {
			defer tr.Un(tr.Trace("inner"))
			tr.Message("hello")
		}()
	}()
	printer.PrintTrace(tr, true, os.Stdout)
	// Output:
	// > outer
	// ..> inner
	// ....@ [inner] hello
	// ..< inner
	// < outer
}

func ExampleTraceCtx() {
	tr := trace.NewTracer()
	ctx := trace.WithTracer(context.Background(), tr)
	work(ctx)
	printer.PrintTrace(tr, true, os.Stdout)
	// Output:
	// > work
	// ..@ [work] computing
	// < work
}

func work(ctx context.Context) {
	defer trace.TraceCtx(ctx)()
	trace.MessageStrCtx(ctx, "computing")
}

func ExampleIsTracing() {
	// Skip expensive arg construction when no tracer is attached.
	ctx := context.Background() // no tracer
	if trace.IsTracing(ctx) {
		trace.MessagefCtx(ctx, "x=%d", 42)
	}
	// Output:
}

func ExampleNewTracerWithTime() {
	tr := trace.NewTracerWithTime()
	defer tr.Un(tr.Trace("work"))
	// nodes carry timestamps now; ordinary printing still works
	_ = tr
	// Output:
}

func ExampleTracer_Reset() {
	tr := trace.NewTracer()
	for i := 0; i < 2; i++ {
		tr.Reset()
		func() {
			defer tr.Un(tr.Trace("cycle"))
		}()
	}
	// Output:
}

func ExampleTracer_SetMessagesEnabled() {
	tr := trace.NewTracer()
	tr.SetMessagesEnabled(false)
	func() {
		defer tr.Un(tr.Trace("scope"))
		tr.Message("dropped")
		tr.SetMessagesEnabled(true)
		tr.Message("kept")
	}()
	printer.PrintTrace(tr, true, os.Stdout)
	// Output:
	// > scope
	// ..@ [scope] kept
	// < scope
}
