package printer_test

import (
	"os"

	"github.com/lczyk/trace"
	"github.com/lczyk/trace/printer"
)

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
