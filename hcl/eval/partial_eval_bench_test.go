// Copyright 2023 Mineiros GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eval_test

import (
	"os"
	"strings"
	"testing"

	hhcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/mineiros-io/terramate/globals"
	"github.com/mineiros-io/terramate/hcl/eval"
	"github.com/mineiros-io/terramate/hcl/info"
	"github.com/mineiros-io/terramate/project"
	"github.com/mineiros-io/terramate/stdlib"
	"github.com/mineiros-io/terramate/test/sandbox"
	"github.com/zclconf/go-cty/cty"
)

func setupContext(b *testing.B) *eval.Context {
	s := sandbox.New(b)
	builtinInfo := eval.Info{
		Scope: project.NewPath("/"),
		DefinedAt: info.NewRange(s.RootDir(), hhcl.Range{
			Start: hhcl.InitialPos,
			End:   hhcl.InitialPos,
		}),
	}
	ctx := eval.New(
		globals.NewResolver(
			s.Config().Tree(),
			eval.NewValStmt(eval.NewRef("global", "true"), cty.True, builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "false"), cty.False, builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "number"), cty.NumberFloatVal(3.141516), builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "string"), cty.StringVal("terramate"), builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "list"), cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}), builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "strings"), cty.ListVal([]cty.Value{
				cty.StringVal("terramate"),
				cty.StringVal("is"),
				cty.StringVal("fun"),
			}), builtinInfo),
			eval.NewValStmt(eval.NewRef("global", "obj"), cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(0),
				"b": cty.ListVal([]cty.Value{cty.StringVal("terramate")}),
			}), builtinInfo),
		))
	ctx.SetFunctions(stdlib.Functions(ctx, os.TempDir()))
	return ctx
}

func BenchmarkPartialEvalComplex(b *testing.B) {
	b.StopTimer()
	ctx := setupContext(b)

	exprBytes := []byte(`[
		{
			a = "prefix ${tm_upper(global.string)} ${global.number} suffix"
			b = [0, 1, global.true, global.false, global.number, global.string, global.list, global.obj]
			c = {
				a = tm_floor(global.number) == 3 ? tm_upper(global.string) : tm_title(global.string)
				b = 10*global.number+global.number / 2+3
			}
			e = tm_concat(global.list, [tm_max(21, 8, 13, 3, 1, 5, 1, 2)])
		},
		{
			a = "prefix ${tm_upper(global.string)} ${global.number} suffix"
			b = [0, 1, global.true, global.false, global.number, global.string, global.list, global.obj]
			c = {
				a = tm_floor(global.number) == 3 ? tm_upper(global.string) : tm_title(global.string)
				b = 10*global.number+global.number / 2+3
			}
			e = tm_concat(global.list, [tm_max(21, 8, 13, 3, 1, 5, 1, 2)])
		},
		{
			a = "prefix ${tm_upper(global.string)} ${global.number} suffix"
			b = [0, 1, global.true, global.false, global.number, global.string, global.list, global.obj]
			c = {
				a = tm_floor(global.number) == 3 ? tm_upper(global.string) : tm_title(global.string)
				b = 10*global.number+global.number / 2+3
			}
			e = tm_concat(global.list, [tm_max(21, 8, 13, 3, 1, 5, 1, 2)])
		},
		{
			a = "prefix ${tm_upper(global.string)} ${global.number} suffix"
			b = [0, 1, global.true, global.false, global.number, global.string, global.list, global.obj]
			c = {
				a = tm_floor(global.number) == 3 ? tm_upper(global.string) : tm_title(global.string)
				b = 10*global.number+global.number / 2+3
			}
			e = tm_concat(global.list, [tm_max(21, 8, 13, 3, 1, 5, 1, 2)])
		},
		{
			a = "prefix ${tm_upper(global.string)} ${global.number} suffix"
			b = [0, 1, global.true, global.false, global.number, global.string, global.list, global.obj]
			c = {
				a = tm_floor(global.number) == 3 ? tm_upper(global.string) : tm_title(global.string)
				b = 10*global.number+global.number / 2+3
			}
			e = tm_concat(global.list, [tm_max(21, 8, 13, 3, 1, 5, 1, 2)])
		},
	]`)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		expr, diags := hclsyntax.ParseExpression(exprBytes, "<bench>", hhcl.InitialPos)
		if diags.HasErrors() {
			b.Fatalf(diags.Error())
		}
		_, err := ctx.PartialEval(expr)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func BenchmarkPartialEvalSmallString(b *testing.B) {
	b.StopTimer()
	ctx := setupContext(b)

	exprBytes := []byte(`"terramate is fun"`)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		expr, diags := hclsyntax.ParseExpression(exprBytes, "<bench>", hhcl.InitialPos)
		if diags.HasErrors() {
			b.Fatalf(diags.Error())
		}
		_, err := ctx.PartialEval(expr)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func BenchmarkPartialEvalHugeString(b *testing.B) {
	b.StopTimer()
	ctx := setupContext(b)

	exprBytes := []byte(`"` + strings.Repeat(`terramate is fun\n`, 1000) + `"`)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		expr, diags := hclsyntax.ParseExpression(exprBytes, "<bench>", hhcl.InitialPos)
		if diags.HasErrors() {
			b.Fatalf(diags.Error())
		}
		_, err := ctx.PartialEval(expr)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func BenchmarkPartialEvalHugeInterpolatedString(b *testing.B) {
	b.StopTimer()
	ctx := setupContext(b)

	exprBytes := []byte(`"` + strings.Repeat(`${global.string} is fun\n`, 1000) + `"`)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		expr, diags := hclsyntax.ParseExpression(exprBytes, "<bench>", hhcl.InitialPos)
		if diags.HasErrors() {
			b.Fatalf(diags.Error())
		}
		_, err := ctx.PartialEval(expr)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func BenchmarkPartialEvalObject(b *testing.B) {
	b.StopTimer()
	ctx := setupContext(b)

	exprBytes := []byte(`{
		a = 1
		b = [0, 1, 2, 3]
		c = [global.number, global.string]
		d = [global.list]	
	}`)

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		expr, diags := hclsyntax.ParseExpression(exprBytes, "<bench>", hhcl.InitialPos)
		if diags.HasErrors() {
			b.Fatalf(diags.Error())
		}
		_, err := ctx.PartialEval(expr)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}
