package trace_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lczyk/trace"
	assert "github.com/lczyk/trace/internal/muert"
	"github.com/lczyk/trace/printer"
)

// Trace with (format, args...) splats string args into Sprintf.
func TestTraceFormatArgs(t *testing.T) {
	tr := trace.NewTracer()
	tr.Un(tr.Trace("a.%s.b", "c"))
	tr.Un(tr.Trace("%s:%s", "x", "y"))
	var names []string
	w, err := tr.ToWalkable()
	assert.NoError(t, err)
	_ = w.Walk(func(n trace.Node) error {
		if e, ok := n.(*trace.Enter); ok {
			if e.Name() != trace.START_NODE {
				names = append(names, e.Name())
			}
		}
		return nil
	})
	assert.Equal(t, len(names), 2)
	assert.Equal(t, names[0], "a.c.b")
	assert.Equal(t, names[1], "x:y")
}

// Done is idempotent: calling twice does not append a second END exit.
func TestDoneIdempotent(t *testing.T) {
	tr := trace.NewTracer()
	tr.Un(tr.Trace("x"))
	tr.Done()
	n1 := tr.Len()
	tr.Done()
	tr.Done()
	assert.Equal(t, tr.Len(), n1)
}

// ToWalkable can be called repeatedly; implicit Done only runs once.
func TestToWalkableIdempotent(t *testing.T) {
	tr := trace.NewTracer()
	tr.Un(tr.Trace("x"))
	_, err := tr.ToWalkable()
	assert.NoError(t, err)
	n1 := tr.Len()
	_, err = tr.ToWalkable()
	assert.NoError(t, err)
	assert.Equal(t, tr.Len(), n1)
}

// SetMessagesEnabled toggled mid-trace gates only intervening messages.
func TestMessagesEnabledMidTrace(t *testing.T) {
	tr := trace.NewTracer()
	tr.Message("a")
	tr.SetMessagesEnabled(false)
	tr.Message("b")
	tr.Messagef("c=%d", 1)
	tr.SetMessagesEnabled(true)
	tr.Message("d")
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 2)
	assert.Equal(t, msgs[0].Message, "a")
	assert.Equal(t, msgs[1].Message, "d")
}

// Messagef with mismatched format/args produces fmt's default error
// rendering but does not panic.
func TestMessagefMismatch(t *testing.T) {
	tr := trace.NewTracer()
	tr.Messagef("%d %s", "not-an-int") // missing arg + wrong type
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 1)
	if !strings.Contains(msgs[0].Message, "!") {
		t.Fatalf("expected fmt error rendering, got %q", msgs[0].Message)
	}
}

// Printer tolerates names containing newlines without panicking. The
// rendered output won't be structured, but the call must not crash.
func TestPrinterWeirdNames(t *testing.T) {
	tr := trace.NewTracer()
	tr.Un(tr.Trace("line\nbreak"))
	tr.Message("msg\nwith\nnewlines")
	var buf bytes.Buffer
	printer.PrintTrace(tr, true, &buf)
	if buf.Len() == 0 {
		t.Fatal("expected some output")
	}
}

// Trace name from Here() lookup with no explicit name.
func TestTraceImplicitName(t *testing.T) {
	tr := trace.NewTracer()
	implicit(tr)
	w, err := tr.ToWalkable()
	assert.NoError(t, err)
	var found bool
	_ = w.Walk(func(n trace.Node) error {
		if e, ok := n.(*trace.Enter); ok && e.Name() == "implicit" {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatal("expected Enter node named 'implicit'")
	}
}

func implicit(tr trace.Tracer) {
	defer tr.Un(tr.Trace())
}

// Walk over an empty (just-created) tracer yields only the START node.
func TestWalkEmpty(t *testing.T) {
	tr := trace.NewTracer()
	w, err := tr.ToWalkable()
	assert.NoError(t, err)
	count := 0
	_ = w.Walk(func(n trace.Node) error {
		count++
		return nil
	})
	// START enter + END exit = 2
	assert.Equal(t, count, 2)
}

// Message.ParentName returns "" for root-scope messages.
func TestMessageParentNameRoot(t *testing.T) {
	tr := trace.NewTracer()
	tr.Message("root")
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].ParentName(), "")
}

// Default tracer leaves all Time() values as zero.
func TestDefaultTracerNoTimestamps(t *testing.T) {
	tr := trace.NewTracer()
	defer tr.Un(tr.Trace("x"))
	tr.Message("hi")
	w, err := tr.ToWalkable()
	assert.NoError(t, err)
	_ = w.Walk(func(n trace.Node) error {
		switch n := n.(type) {
		case *trace.Enter:
			if !n.Time().IsZero() {
				t.Fatalf("expected zero time on default Enter, got %v", n.Time())
			}
		case *trace.Exit:
			if !n.Time().IsZero() {
				t.Fatalf("expected zero time on default Exit, got %v", n.Time())
			}
		case *trace.Message:
			if !n.Time().IsZero() {
				t.Fatalf("expected zero time on default Message, got %v", n.Time())
			}
		}
		return nil
	})
}

// NewTracerWithTime stamps every node and timestamps are monotonic.
func TestTracerWithTimeStamps(t *testing.T) {
	tr := trace.NewTracerWithTime()
	ex := tr.Trace("x")
	tr.Message("hi")
	time.Sleep(time.Millisecond)
	tr.Un(ex)
	w, err := tr.ToWalkable()
	assert.NoError(t, err)
	var times []time.Time
	_ = w.Walk(func(n trace.Node) error {
		switch n := n.(type) {
		case *trace.Enter:
			times = append(times, n.Time())
		case *trace.Exit:
			times = append(times, n.Time())
		case *trace.Message:
			times = append(times, n.Time())
		}
		return nil
	})
	for i, ts := range times {
		if ts.IsZero() {
			t.Fatalf("node %d had zero time", i)
		}
	}
	for i := 1; i < len(times); i++ {
		if times[i].Before(times[i-1]) {
			t.Fatalf("non-monotonic at %d: %v before %v", i, times[i], times[i-1])
		}
	}
}

// Reset drops recorded nodes and lets the tracer be reused.
func TestReset(t *testing.T) {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("first"))
		tr.Message("a")
	}()
	assert.Equal(t, len(tr.Messages()), 1)

	tr.Reset()
	assert.Equal(t, tr.Len(), 1) // just START

	func() {
		defer tr.Un(tr.Trace("second"))
		tr.Message("b")
	}()
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].Message, "b")
	assert.Equal(t, msgs[0].ParentName(), "second")
}

// Reset preserves the message-enabled gate.
func TestResetKeepsGate(t *testing.T) {
	tr := trace.NewTracer()
	tr.SetMessagesEnabled(false)
	tr.Reset()
	if tr.MessagesEnabled() {
		t.Fatal("expected gate to survive Reset")
	}
}

// MaybeWithTracer respects the enabled flag.
func TestMaybeWithTracer(t *testing.T) {
	ctx := context.Background()
	off := trace.MaybeWithTracer(ctx, false)
	if trace.IsTracing(off) {
		t.Fatal("expected no tracer when disabled")
	}
	on := trace.MaybeWithTracer(ctx, true)
	if !trace.IsTracing(on) {
		t.Fatal("expected tracer when enabled")
	}
}

// Message.ParentName returns the immediate enclosing scope name.
func TestMessageParentNameScope(t *testing.T) {
	tr := trace.NewTracer()
	defer tr.Un(tr.Trace("outer"))
	tr.Message("inside")
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].ParentName(), "outer")
}
