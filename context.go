package trace

import (
	"context"
)

type tracerKey struct{}

func WithTracer(ctx context.Context, tracer Tracer) context.Context {
	return context.WithValue(ctx, tracerKey{}, tracer)
}

// MaybeWithTracer attaches a fresh tracer to ctx iff enabled is true,
// otherwise returns ctx unchanged. Convenience for the common
// conditional-trace pattern:
//
//	ctx = trace.MaybeWithTracer(ctx, opts.Trace)
func MaybeWithTracer(ctx context.Context, enabled bool) context.Context {
	if !enabled {
		return ctx
	}
	return WithTracer(ctx, NewTracer())
}

func GetTracer(ctx context.Context) Tracer {
	if ctx == nil {
		// no context, no tracer
		return nil
	}
	tracer, ok := ctx.Value(tracerKey{}).(Tracer)
	if !ok {
		return nil
	}
	return tracer
}

// WITHOUT CONTEXT:
//
//	if tracer != nil {
//	    defer tracer.Un(tracer.Trace(trace.Here()))
//	}
//
// WITH CONTEXT:
// NOTE the extra `()`
//
//	defer trace.TraceCtx(ctx)()
//
//go:noinline
func TraceCtx(ctx context.Context, where ...string) func() {
	tracer := GetTracer(ctx)

	if tracer == nil {
		return func() {}
	} else {
		// we have a tracer
		where_str := whereToString(where...)
		t := tracer.Trace(where_str)
		return func() {
			tracer.Un(t)
		}
	}
}

//go:noinline
func HereCtx(ctx context.Context) string {
	tracer := GetTracer(ctx)
	if tracer != nil {
		return here(callerName(1))
	}
	return ""
}

// MessageCtx records a message on the tracer attached to ctx, if any.
//
// Performance note: the variadic ...any parameter forces the args slice
// to be allocated at the callsite even when no tracer is attached
// (~16B / 1 alloc per call). For hot paths that pass a single string,
// prefer [MessageStrCtx], which is non-variadic and stays alloc-free in
// the no-tracer case. For mixed values, guard the call with
// [IsTracing].
func MessageCtx(ctx context.Context, args ...any) {
	tracer := GetTracer(ctx)
	if tracer != nil {
		tracer.Message(args...)
	}
}

// MessagefCtx records a printf-style message on the tracer attached to
// ctx, if any.
//
// Performance note: variadic ...any forces the args slice to be
// allocated at the callsite even when no tracer is attached
// (~24B / 1 alloc per call). For hot paths, guard with [IsTracing] to
// skip both the alloc and the Sprintf.
func MessagefCtx(ctx context.Context, format string, args ...any) {
	tracer := GetTracer(ctx)
	if tracer != nil {
		tracer.Messagef(format, args...)
	}
}

// IsTracing reports whether ctx carries an active tracer. Use to guard
// hot-path Message/Messagef calls and skip the variadic-args alloc when
// tracing is off:
//
//	if trace.IsTracing(ctx) { trace.MessagefCtx(ctx, "x=%d", x) }
func IsTracing(ctx context.Context) bool {
	return GetTracer(ctx) != nil
}

// MessageStrCtx is a non-variadic fast path for single-string messages.
// Avoids the variadic-args slice alloc paid at every MessageCtx callsite
// even when no tracer is present.
func MessageStrCtx(ctx context.Context, msg string) {
	tracer := GetTracer(ctx)
	if tracer != nil {
		tracer.Message(msg)
	}
}
