// Analyzer flags calls to github.com/lczyk/trace.MessageCtx whose only
// data argument is a single string. Such callsites pay an unnecessary
// variadic-args alloc (~16B / 1 alloc per call, even when no tracer is
// attached to the context). MessageStrCtx is the non-variadic
// equivalent and stays alloc-free on the no-tracer path.
package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const tracePkgPath = "github.com/lczyk/trace"

var Analyzer = &analysis.Analyzer{
	Name:     "tracestrctx",
	Doc:      "suggest trace.MessageStrCtx for single-string calls to trace.MessageCtx",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		// resolve the selected identifier; this gives us the package + name
		// of the function being called, robust to import-aliasing.
		obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
		if !ok || obj.Pkg() == nil {
			return
		}
		if obj.Pkg().Path() != tracePkgPath || obj.Name() != "MessageCtx" {
			return
		}
		// MessageCtx(ctx, args...) -- we want exactly 2 args, the second
		// being string-typed.
		if len(call.Args) != 2 {
			return
		}
		tv, ok := pass.TypesInfo.Types[call.Args[1]]
		if !ok {
			return
		}
		basic, ok := tv.Type.Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.String {
			return
		}
		pass.Reportf(call.Pos(),
			"trace.MessageCtx with a single string arg pays a variadic-args alloc; use trace.MessageStrCtx instead")
	})

	return nil, nil
}
