package printer_test

import (
	"os"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

func ExamplePrettyPrintMessage_multiline() {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("a"))
		func() {
			defer tr.Un(tr.Trace("b"))
			func() {
				defer tr.Un(tr.Trace("c"))
				tr.Message("hi")
			}()
		}()
	}()
	for _, m := range tr.Messages() {
		printer.PrettyPrintMessage(m, printer.MULTILINE)
	}
	// Output:
	// a
	// \_b
	//  \_c: hi
}

func ExamplePrintChromeTrace() {
	tr := trace.NewTracerWithTime()
	defer tr.Un(tr.Trace("work"))
	// pipe os.Stdout into a .json file, then load in chrome://tracing
	// or perfetto.dev/ui to view the call tree as a flame graph.
	_ = printer.PrintChromeTrace(tr, os.Stdout)
}

func ExamplePrintNDJSON() {
	tr := trace.NewTracer()
	func() {
		defer tr.Un(tr.Trace("scope"))
		tr.Message("hi")
	}()
	_ = printer.PrintNDJSON(tr, os.Stdout)
	// Output:
	// {"kind":"enter","name":"<START>"}
	// {"kind":"enter","name":"scope"}
	// {"kind":"message","scope":"scope","message":"hi"}
	// {"kind":"exit","name":"scope"}
	// {"kind":"exit","name":"<END>"}
}
