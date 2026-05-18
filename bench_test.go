package trace_test

import (
	"context"
	"io"
	"testing"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

func BenchmarkTraceUn(b *testing.B) {
	tr := trace.NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ex := tr.Trace("scope")
		tr.Un(ex)
	}
}

func BenchmarkTraceHereUn(b *testing.B) {
	tr := trace.NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Un(tr.Trace(trace.Here()))
	}
}

func BenchmarkMessage(b *testing.B) {
	tr := trace.NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Message("hello")
	}
}

func BenchmarkMessagef(b *testing.B) {
	tr := trace.NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Messagef("k=%d", i)
	}
}

func BenchmarkMessageDisabled(b *testing.B) {
	tr := trace.NewTracer()
	tr.SetMessagesEnabled(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Message("hello")
	}
}

func BenchmarkMessagefDisabled(b *testing.B) {
	tr := trace.NewTracer()
	tr.SetMessagesEnabled(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Messagef("k=%d", i)
	}
}

func recurse(tr trace.Tracer, depth int) {
	defer tr.Un(tr.Trace("r"))
	if depth > 0 {
		recurse(tr, depth-1)
	}
}

func BenchmarkDeepNesting(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tr := trace.NewTracer()
		recurse(tr, 64)
	}
}

func BenchmarkReallyDeepNesting(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tr := trace.NewTracer()
		recurse(tr, 1024)
	}
}

func BenchmarkToWalkable(b *testing.B) {
	tr := trace.NewTracer()
	for i := 0; i < 100; i++ {
		ex := tr.Trace("s")
		tr.Message("m")
		tr.Un(ex)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tr.ToWalkable()
	}
}

func BenchmarkSprintTrace(b *testing.B) {
	tr := trace.NewTracer()
	for i := 0; i < 50; i++ {
		ex := tr.Trace("s")
		tr.Message("m")
		tr.Un(ex)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = printer.SprintTrace(tr, true)
	}
}

func BenchmarkPrintTraceDiscard(b *testing.B) {
	tr := trace.NewTracer()
	for i := 0; i < 50; i++ {
		ex := tr.Trace("s")
		tr.Message("m")
		tr.Un(ex)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		printer.PrintTrace(tr, true, io.Discard)
	}
}

func BenchmarkTraceCtxNoTracer(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := trace.TraceCtx(ctx)
		done()
	}
}

func BenchmarkTraceCtxWithTracer(b *testing.B) {
	tr := trace.NewTracer()
	ctx := trace.WithTracer(context.Background(), tr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := trace.TraceCtx(ctx, "scope")
		done()
	}
}

func BenchmarkMessageCtxNoTracer(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace.MessageCtx(ctx, "hello")
	}
}

func BenchmarkMessagefCtxNoTracer(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace.MessagefCtx(ctx, "k=%d", i)
	}
}
