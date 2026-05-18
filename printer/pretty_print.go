package printer

import (
	"fmt"
	"strings"

	"github.com/lczyk/trace"
)

type Flavour int

const (
	ONELINE Flavour = iota
	MULTILINE
)

func PrettySprintMessage(message trace.Message, flavour ...Flavour) string {
	if len(flavour) == 0 {
		flavour = append(flavour, ONELINE)
	}

	var out string
	switch flavour[0] {
	case ONELINE:
		out = prettyPrintOneline(message)
	case MULTILINE:
		out = prettyPrintMultiline(message)
	default:
		panic("Unknown flavour")
	}
	return out
}

func PrettyPrintMessage(message trace.Message, flavour ...Flavour) {
	fmt.Print(PrettySprintMessage(message, flavour...))
}

// flavours

func prettyPrintOneline(message trace.Message) string {
	return message.String()
}

func prettyPrintMultiline(message trace.Message) string {
	// stack[0] is the immediate enclosing scope, stack[len-1] the outermost
	// ancestor. Print outermost first so the visual nesting reads top-down
	// like a real call tree.
	var out strings.Builder
	stack := message.Stack()
	n := len(stack)
	for depth := 0; depth < n; depth++ {
		// stack index goes from outermost (n-1) down to innermost (0)
		j := n - 1 - depth
		if depth > 0 {
			out.WriteString(strings.Repeat(" ", depth-1))
			out.WriteString("\\_") // ASCII tree branch
		}
		out.WriteString(stack[j])
		if depth < n-1 {
			out.WriteString("\n")
		}
	}
	out.WriteString(fmt.Sprintf(": %s\n", message.Message))
	return out.String()
}
