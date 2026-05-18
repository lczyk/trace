// micro version of assert to copy/paste into other packages `internal`.
//
// Intentional divergences from github.com/lczyk/assert:
//   - Subset API: only That, Equal, NotEqual, NoError, Error. No EqualCmp,
//     EqualArrays, EqualMaps, Type, Panic, etc.
//   - Error(err, expected) does substring match (strings.Contains), not regex.
//   - NoError uses the generic "assertion failed" default message.
//   - No dependency on the compare subpackage.
package muert

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// License of github.com/lczyk/assert, embedded verbatim so a copy/pasted
// muert.go carries its origin license with it.
const License = `MIT License

Copyright (c) 2025 Marcin Konowalczyk @lczyk

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`

// Check that the predicate is true, otherwise it fail the test.
func That(t testing.TB, predicate bool, args ...any) { t.Helper(); assert(t, 2, predicate, args) }

// Check that two comparable values are equal, otherwise it fail the test.
func Equal[T comparable](t testing.TB, a T, b T) {
	t.Helper()
	assert(t, 2, a == b, []any{"expected '%v' (%T) == '%v' (%T)", a, a, b, b})
}

// Check that two comparable values are not equal, otherwise it fail the test.
func NotEqual[T comparable](t testing.TB, a T, b T) {
	t.Helper()
	assert(t, 2, a != b, []any{"expected '%v' (%T) != '%v' (%T)", a, a, b, b})
}

// Check that an error is nil.
func NoError(t testing.TB, err error, args ...any) { t.Helper(); assert(t, 2, err == nil, args) }

// Check that the error is not nil and contains the expected message.
func Error(t testing.TB, err error, expected string, args ...any) {
	t.Helper()
	if err == nil {
		assert(t, 2, false, []any{"expected error containing '%s', got nil", expected})
		return
	}
	errs := err.Error()
	assert(t, 2, strings.Contains(errs, expected), []any{
		"expected error to contain '%s', got '%s' (%T): %v",
		expected, errs, err, args,
	})
}

func get_parent_info(N int) (string, int) {
	parent, _, _, _ := runtime.Caller(1 + N)
	return runtime.FuncForPC(parent).FileLine(parent)
}

// convert 'args ...any' to the assertion message
// internal utility so we don't use variadics to make the calls a bit more consistent
func args_to_message(args []any) string {
	var msg string = "assertion failed"
	if len(args) > 0 {
		switch a := args[0].(type) {
		case string:
			msg = fmt.Sprintf(a, args[1:]...)
		default:
			msg = fmt.Sprintf("%v", args)
		}
	}
	return msg
}

func assert(t testing.TB, N int, predicate bool, args []any) {
	t.Helper()
	if !predicate {
		file, line := get_parent_info(N)
		t.Errorf(args_to_message(args)+" in %s:%d", file, line)
	}
}
