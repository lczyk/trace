package trace_test

import (
	"bytes"
	"strings"
	"testing"

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

// Message.ParentName returns the immediate enclosing scope name.
func TestMessageParentNameScope(t *testing.T) {
	tr := trace.NewTracer()
	defer tr.Un(tr.Trace("outer"))
	tr.Message("inside")
	msgs := tr.Messages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].ParentName(), "outer")
}
