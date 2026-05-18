package trace_test

import (
	"fmt"
	"testing"

	"github.com/lczyk/trace"
	assert "github.com/lczyk/trace/internal/muert"
	"github.com/lczyk/trace/printer"
)

func TestNewTracer(t *testing.T) {
	tracer := trace.NewTracer()
	assert.Equal(t, len(tracer.Messages()), 0)
}

////////////////////////////////////////////////////////////////////////////////

// called third
func third(tr trace.Tracer) {
	defer tr.Un(tr.Trace(trace.Here()))
	tr.Message("Second")
}

// called second
func second(tr trace.Tracer) {
	defer tr.Un(tr.Trace(trace.Here()))
	third(tr)
	tr.Message("Third") // third since _third is called before
}

// called first
func first(tr trace.Tracer) {
	defer tr.Un(tr.Trace(trace.Here()))
	tr.Message("First")
	second(tr)
}

func TestTraceNormal(t *testing.T) {
	tracer := trace.NewTracer()
	first(tracer)
	tracer.Done()
	messages := tracer.Messages()
	assert.Equal(t, len(messages), 3)
	for _, message := range messages {
		printer.PrettyPrintMessage(message, printer.MULTILINE)
	}
}

////////////////////////////////////////////////////////////////////////////////

func first_gap(tr trace.Tracer) {
	defer tr.Un(tr.Trace(trace.Here()))
	tr.Message("First")
	second_gap(tr)
}

func second_gap(tr trace.Tracer) {
	// oops! we do not register the intermediate function with the tracer!
	// defer tr.Un(tr.Trace())
	tr.Message("Second")
	third_gap(tr)
}

func third_gap(tr trace.Tracer) {
	// and we did not register this one either!
	// defer tr.Un(tr.Trace())
	tr.Message("Third")
}

func TestTraceGap(t *testing.T) {
	tracer := trace.NewTracer()
	first_gap(tracer)
	tracer.Done()
	messages := tracer.Messages()
	assert.Equal(t, len(messages), 3)
	for _, message := range messages {
		fmt.Printf("%s\n", message)
		// printer.PrettyPrint(message, printer.MULTILINE)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestTraceWithout(t *testing.T) {
	tracer := trace.NewTracer()
	tracer.Message("First")
	tracer.Done()
	messages := tracer.Messages()
	assert.Equal(t, len(messages), 1)
	for _, message := range messages {
		printer.PrettyPrintMessage(message, printer.MULTILINE)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestTraceToWalkable(t *testing.T) {
	tracer := trace.NewTracer()
	first(tracer)
	// ToWalkable now Dones implicitly; no error path expected
	walkable, err := tracer.ToWalkable()
	assert.NoError(t, err)
	fmt.Println("Walkable:")
	_ = walkable.Walk(func(node trace.Node) error {
		fmt.Println(node)
		return nil
	})
}

func TestSetMessagesEnabled(t *testing.T) {
	tracer := trace.NewTracer()
	tracer.Message("on1")
	tracer.SetMessagesEnabled(false)
	tracer.Message("off")
	tracer.Messagef("off %d", 2)
	tracer.SetMessagesEnabled(true)
	tracer.Messagef("on %d", 2)
	assert.Equal(t, len(tracer.Messages()), 2)
}

func TestMessagef(t *testing.T) {
	tracer := trace.NewTracer()
	tracer.Messagef("%s=%d", "x", 42)
	msgs := tracer.Messages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].Message, "x=42")
}
