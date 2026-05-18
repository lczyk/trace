# trace

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/lczyk/trace)
![GitHub Tag](https://img.shields.io/github/v/tag/lczyk/trace?label=release)
[![lint_and_test](https://github.com/lczyk/trace/actions/workflows/lint_and_test.yml/badge.svg)](https://github.com/lczyk/trace/actions/workflows/lint_and_test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lczyk/trace.svg)](https://pkg.go.dev/github.com/lczyk/trace)
[![Go Report Card](https://goreportcard.com/badge/github.com/lczyk/trace)](https://goreportcard.com/report/github.com/lczyk/trace)

structured call-tree tracing for go. records function enter/exit + inline messages, prints the tree, optional context-carried so callers without a tracer pay near-zero cost.

```
go get github.com/lczyk/trace
```

## quick start

direct usage:

```go
tr := trace.NewTracer()
parse(tr)
printer.PrintTrace(tr, true, os.Stdout)

func parse(tr trace.Tracer) {
    defer tr.Un(tr.Trace(trace.Here()))
    tr.Message("starting")
    lex(tr)
}
```

context usage (callers without a tracer pay nothing):

```go
ctx := trace.WithTracer(context.Background(), trace.NewTracer())
parse(ctx)

func parse(ctx context.Context) {
    defer trace.TraceCtx(ctx)()
    trace.MessageStrCtx(ctx, "starting")
}
```

output:

```
> parse
..@ [parse] starting
..> lex
....@ [lex] token INT
..< lex
< parse
```

## api shape

- `NewTracer()` -- create a tracer.
- `Trace(name...) *Exit` / `Un(*Exit)` -- paired enter/exit; `defer tr.Un(tr.Trace(...))`.
- `Here()` -- auto-name from caller info.
- `Message`, `Messagef` -- inline messages.
- `SetMessagesEnabled(bool)` -- runtime gate. "structure-only" / "no-messages" mode: when off, `Message` / `Messagef` become no-ops while enter/exit nodes still record. Toggle freely mid-trace to silence noisy sections (e.g. per-token logging inside a parser) without losing the call-tree shape.
- `ToWalkable()` -- finalises the trace (idempotent), returns an iterable.

### context helpers

- `WithTracer(ctx, tr)` / `GetTracer(ctx)`
- `TraceCtx(ctx, name...) func()` -- `defer trace.TraceCtx(ctx)()`
- `MessageCtx`, `MessagefCtx`, `MessageStrCtx`
- `IsTracing(ctx) bool` -- guard hot-path callsites to skip variadic-args alloc

### printer

```go
printer.PrintTrace(tr, true, os.Stdout)         // tree to writer
s := printer.SprintTrace(tr, true)              // tree to string
printer.PrettyPrintMessage(msg, printer.MULTILINE)
```

## concurrency

`NewTracer()` returns a tracer that is **not safe for concurrent use**. wrap with `SyncTracer` if multiple goroutines share one:

```go
tr := trace.NewSyncTracer(trace.NewTracer())
```

## linter

`cmd/trace-analyzer` is a `go vet` tool that flags `trace.MessageCtx(ctx, "literal")` callsites and suggests `trace.MessageStrCtx` (no variadic-args alloc on the no-tracer path).

install:

```
go install github.com/lczyk/trace/cmd/trace-analyzer@latest
```

run as a vettool:

```
go vet -vettool=$(which trace-analyzer) ./...
```

example output:

```
parser.go:437:3: trace.MessageCtx with a single string arg pays a variadic-args alloc; use trace.MessageStrCtx instead
```

the analyzer lives in its own submodule (`cmd/trace-analyzer/go.mod`) so the main `trace` package stays dep-free for library consumers.

## demos

```
make demo
```

runs the showcases in `demo/`.

## license

mit. see `LICENCE`.
