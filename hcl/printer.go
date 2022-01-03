// Copyright 2021 Mineiros GmbH
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

package hcl

import (
	"io"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func PrintConfig(w io.Writer, cfg Config) error {
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	if cfg.Terramate != nil {
		tm := cfg.Terramate
		tmBlock := rootBody.AppendNewBlock("terramate", nil)
		tmBody := tmBlock.Body()
		tmBody.SetAttributeValue("required_version", cty.StringVal(tm.RequiredVersion))
	}

	if cfg.Terramate != nil && cfg.Stack != nil {
		rootBody.AppendNewline()
	}

	if cfg.Stack != nil {
		stack := cfg.Stack
		stackBlock := rootBody.AppendNewBlock("stack", nil)
		stackBody := stackBlock.Body()

		if len(stack.After) > 0 {
			stackBody.SetAttributeValue("after", cty.SetVal(listToValue(stack.After)))
		}

		if len(stack.Before) > 0 {
			stackBody.SetAttributeValue("before", cty.SetVal(listToValue(stack.Before)))
		}
	}

	_, err := w.Write(f.Bytes())
	return err
}

func listToValue(list []string) []cty.Value {
	vlist := make([]cty.Value, len(list))
	for i, val := range list {
		vlist[i] = cty.StringVal(val)
	}

	return vlist
}