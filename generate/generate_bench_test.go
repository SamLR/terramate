// Copyright 2022 Mineiros GmbH
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

package generate_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/madlambda/spells/assert"
	"github.com/mineiros-io/terramate/generate"
	"github.com/mineiros-io/terramate/project"
	"github.com/mineiros-io/terramate/test/sandbox"

	. "github.com/mineiros-io/terramate/test/hclwrite/hclutils"
)

func BenchmarkGenerateFlatStacks_50(b *testing.B) {
	const nstacks = 50
	s := sandbox.New(b)
	createStacks(s, nstacks)

}

type benchmark struct {
	stacks   int
	asserts  int
	genhcl   int
	genfiles int
	globals  int
}

func (bm benchmark) String() string {
	return fmt.Sprintf("stacks=%d asserts=%d genhcl=%d genfiles=%d globals=%d",
		bm.stacks, bm.asserts, bm.genhcl, bm.genfiles, bm.globals)
}

func (bm benchmark) setup(b *testing.B) sandbox.S {
	b.StopTimer()
	defer b.StartTimer()

	s := sandbox.New(b)
	createStacks(s, bm.stacks)
	createAsserts(s, bm.asserts)

	globals := createGlobals(s, bm.globals)
	createGlobals(s, bm.globals)
	createGenHCLs(s, globals, bm.genhcl)
	createGenFiles(s, globals, bm.genfiles)
	s.Config() // caches config for later use.
	return s
}

func (bm benchmark) assert(b *testing.B, report generate.Report) {
	b.StopTimer()
	defer b.StartTimer()

	assert.EqualInts(b, bm.stacks, len(report.Successes))
	assert.EqualInts(b, 0, len(report.Failures))

	for _, success := range report.Successes {
		assert.EqualInts(b, bm.genhcl+bm.genfiles, len(success.Created))
		assert.EqualInts(b, 0, len(success.Changed))
		assert.EqualInts(b, 0, len(success.Deleted))
	}
}

func (bm benchmark) run(b *testing.B) {
	s := bm.setup(b)
	report := generate.Do(s.Config(), project.NewPath("/modules"), nil)
	bm.assert(b, report)
}

func createStacks(s sandbox.S, stacks int) {
	for i := 0; i < stacks; i++ {
		s.CreateStack(fmt.Sprintf("stacks/stack-%d", i))
	}
}

func createGlobals(s sandbox.S, nglobals int, expr string) []string {
	builder := Globals()
	globalsNames := make([]string, nglobals)

	for i := 0; i < nglobals; i++ {
		name := fmt.Sprintf("val%d", i)
		globalsNames[i] = name
		builder.AddExpr(name, expr)
	}

	s.RootEntry().CreateFile("globals.tm", builder.String())

	return globalsNames
}

func createGenHCLs(s sandbox.S, globals []string, genhcls int) {
	for i := 0; i < genhcls; i++ {
		genhclDoc := GenerateHCL()
		genhclDoc.AddLabel(fmt.Sprintf("gen/%d.hcl", i))

		content := Content()
		for j, global := range globals {
			content.AddExpr(
				fmt.Sprintf("val%d%d", i, j),
				"global."+global)
		}

		genhclDoc.AddBlock(content)

		s.RootEntry().CreateFile(
			fmt.Sprintf("genhcl%d.tm", i),
			genhclDoc.String(),
		)
	}
}

func createGenFiles(s sandbox.S, globals []string, genfiles int) {
	for i := 0; i < genfiles; i++ {
		genfileDoc := GenerateFile()
		genfileDoc.AddLabel(fmt.Sprintf("gen/%d.txt", i))

		content := make([]string, len(globals))

		for j, global := range globals {
			content[j] = fmt.Sprintf("val%d%d=${global.%s}", i, j, global)
		}

		genfileDoc.AddString("content", strings.Join(content, ","))

		s.RootEntry().CreateFile(
			fmt.Sprintf("genfile%d.tm", i),
			genfileDoc.String(),
		)
	}
}

func createAsserts(s sandbox.S, asserts int) {
	for i := 0; i < asserts; i++ {
		assertDoc := Assert()
		assertDoc.AddBoolean("assertion", true)
		assertDoc.AddString("message", fmt.Sprintf("assert %d", i))

		s.RootEntry().CreateFile(
			fmt.Sprintf("assert%d.tm", i),
			assertDoc.String(),
		)
	}
}
