// Package trace records the call tree of a program for inspection and
// printing -- think structured function-entry / function-exit logging
// with optional inline messages.
//
// Two usage shapes:
//
// Direct, with an explicit tracer:
//
//	tr := trace.NewTracer()
//	defer tr.Un(tr.Trace(trace.Here()))
//	tr.Message("hello")
//
// Context-carried, transparent to callers without a tracer:
//
//	ctx := trace.WithTracer(ctx, trace.NewTracer())
//	defer trace.TraceCtx(ctx)()
//	trace.MessageStrCtx(ctx, "hello")
//
// When no tracer is attached to a context, all `*Ctx` helpers are
// near-free no-ops. See [IsTracing] for guarding hot paths whose
// arguments would otherwise alloc at the callsite.
//
// Print captured traces with the sibling printer package.
//
// Tracers are not safe for concurrent use. Wrap with [SyncTracer] if
// multiple goroutines need to share one.
package trace
