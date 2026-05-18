package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/lczyk/trace"
)

// writer wraps io.Writer with a growable dot-indent buffer to avoid
// per-line fmt.Sprintf / strings.Repeat allocs in the hot path.
type writer struct {
	out  io.Writer
	dots []byte // pool of '.' bytes; sliced for indent
	line []byte // scratch buffer for one line
}

func newWriter(out io.Writer) *writer {
	return &writer{
		out:  out,
		dots: make([]byte, 0, 128),
		line: make([]byte, 0, 256),
	}
}

func (w *writer) indent(n int) []byte {
	need := n * 2
	for len(w.dots) < need {
		w.dots = append(w.dots, '.')
	}
	return w.dots[:need]
}

// W writes "<indent><symbol> <name>\n"
func (w *writer) W(symbol byte, indent int, name string) {
	w.line = w.line[:0]
	w.line = append(w.line, w.indent(indent)...)
	w.line = append(w.line, symbol, ' ')
	w.line = append(w.line, name...)
	w.line = append(w.line, '\n')
	w.out.Write(w.line)
}

// W2 writes "<indent><symbol> [<scope>] <name>\n"
func (w *writer) W2(symbol byte, indent int, scope, name string) {
	w.line = w.line[:0]
	w.line = append(w.line, w.indent(indent)...)
	w.line = append(w.line, symbol, ' ', '[')
	w.line = append(w.line, scope...)
	w.line = append(w.line, ']', ' ')
	w.line = append(w.line, name...)
	w.line = append(w.line, '\n')
	w.out.Write(w.line)
}

func NewTracePrinter(
	out io.Writer,
	print_messages bool,
) func(trace.Node) error {
	if out == nil {
		// no output. nothing to do.
		return func(n trace.Node) error { return nil }
	}
	indent := 0
	w := newWriter(out)
	return func(n trace.Node) error {
		switch n := n.(type) {
		case *trace.Enter:
			if n.Name() == trace.START_NODE {
				return nil
			}
			w.W('>', indent, n.Name())
			indent++
		case *trace.Exit:
			if n.Name() == trace.END_NODE {
				return nil
			}
			indent--
			w.W('<', indent, n.Name())
		case *trace.Message:
			if print_messages {
				scope := n.ParentName()
				if scope != "" {
					w.W2('@', indent, scope, n.Message)
				} else {
					w.W('@', indent, n.Message)
				}
			}
		default:
			panic(fmt.Sprintf("unknown node type: %T", n))
		}
		return nil
	}
}

func SprintTrace(
	tracer trace.Tracer,
	print_messages bool,
) string {
	walkable, err := tracer.ToWalkable()
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	_ = walkable.Walk(NewTracePrinter(&out, print_messages))
	// i.stdout.Write([]byte(out.String()))
	return out.String()
}

func PrintTrace(
	tracer trace.Tracer,
	print_messages bool,
	writer io.Writer,
) {
	writer.Write([]byte(SprintTrace(tracer, print_messages)))
}
