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

package ast

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// TokensForExpression generates valid tokens for the given expression.
func TokensForExpression(expr hcl.Expression) hclwrite.Tokens {
	tokens := tokensForExpression(expr)
	tokens = append(tokens, eof())
	return tokens
}

func tokensForExpression(expr hcl.Expression) hclwrite.Tokens {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return literalTokens(e)
	case *hclsyntax.TemplateExpr:
		return templateTokens(e)
	case *hclsyntax.TemplateWrapExpr:
		return templateWrapTokens(e)
	case *hclsyntax.BinaryOpExpr:
		return binOpTokens(e)
	case *hclsyntax.UnaryOpExpr:
		return unaryOpTokens(e)
	case *hclsyntax.TupleConsExpr:
		return tupleTokens(e)
	case *hclsyntax.ParenthesesExpr:
		return parenExprTokens(e)
	case *hclsyntax.ObjectConsExpr:
		return objectTokens(e)
	case *hclsyntax.ObjectConsKeyExpr:
		return objectKeyTokens(e)
	case *hclsyntax.ScopeTraversalExpr:
		return scopeTraversalTokens(e)
	case *hclsyntax.ConditionalExpr:
		return conditionalTokens(e)
	case *hclsyntax.FunctionCallExpr:
		return funcallTokens(e)
	case *hclsyntax.IndexExpr:
		return indexTokens(e)
	case *hclsyntax.ForExpr:
		return forExprTokens(e)
	case *hclsyntax.SplatExpr:
		return splatTokens(e)
	case *hclsyntax.AnonSymbolExpr:
		return anonSplatTokens(e)
	case *hclsyntax.RelativeTraversalExpr:
		return relTraversalTokens(e)
	default:
		panic(fmt.Sprintf("type %T\n", e))
	}
}

func literalTokens(expr *hclsyntax.LiteralValueExpr) hclwrite.Tokens {
	return hclwrite.TokensForValue(expr.Val)
}

func templateTokens(tmpl *hclsyntax.TemplateExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{oquote()}
	for _, part := range tmpl.Parts {
		switch p := part.(type) {
		case *hclsyntax.LiteralValueExpr:
			toks := literalTokens(p)
			if toks[0].Type == hclsyntax.TokenOQuote {
				toks = toks[1 : len(toks)-1]
			}
			tokens = append(tokens, toks...)
		default:
			toks := tokensForExpression(part)
			tokens = append(tokens, interpBegin())
			tokens = append(tokens, toks...)
			tokens = append(tokens, interpEnd())
		}
	}
	tokens = append(tokens, cquote())
	return tokens
}

func templateWrapTokens(tmpl *hclsyntax.TemplateWrapExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{oquote(), interpBegin()}
	tokens = append(tokens, tokensForExpression(tmpl.Wrapped)...)
	tokens = append(tokens, interpEnd(), cquote())
	return tokens
}

func binOpTokens(binop *hclsyntax.BinaryOpExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, tokensForExpression(binop.LHS)...)
	var op hclwrite.Tokens
	switch binop.Op {
	case hclsyntax.OpAdd:
		op = append(op, add())
	case hclsyntax.OpSubtract:
		op = append(op, minus())
	case hclsyntax.OpDivide:
		op = append(op, slash())
	case hclsyntax.OpMultiply:
		op = append(op, star())
	case hclsyntax.OpModulo:
		op = append(op, percent())
	case hclsyntax.OpEqual:
		op = append(op, equal())
	case hclsyntax.OpNotEqual:
		op = append(op, nequal())
	case hclsyntax.OpGreaterThan:
		op = append(op, gtr())
	case hclsyntax.OpLessThan:
		op = append(op, lss())
	case hclsyntax.OpLessThanOrEqual:
		op = append(op, lsseq())
	case hclsyntax.OpGreaterThanOrEqual:
		op = append(op, gtreq())
	case hclsyntax.OpLogicalAnd:
		op = append(op, and())
	case hclsyntax.OpLogicalOr:
		op = append(op, or())
	default:
		panic(fmt.Sprintf("type %T\n", binop.Op))
	}
	tokens = append(tokens, op...)
	tokens = append(tokens, tokensForExpression(binop.RHS)...)
	return tokens
}

func unaryOpTokens(unary *hclsyntax.UnaryOpExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	switch unary.Op {
	case hclsyntax.OpLogicalNot:
		tokens = append(tokens, bang())
	case hclsyntax.OpNegate:
		tokens = append(tokens, minus())
	default:
		panic(fmt.Sprintf("type %T\n", unary.Op))
	}
	tokens = append(tokens, tokensForExpression(unary.Val)...)
	return tokens
}

func parenExprTokens(parenExpr *hclsyntax.ParenthesesExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{oparen()}
	tokens = append(tokens, tokensForExpression(parenExpr.Expression)...)
	tokens = append(tokens, cparen())
	return tokens
}

func tupleTokens(tuple *hclsyntax.TupleConsExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{obrack()}
	for i, expr := range tuple.Exprs {
		tokens = append(tokens, tokensForExpression(expr)...)
		if i+1 != len(tuple.Exprs) {
			tokens = append(tokens, comma())
		}
	}
	tokens = append(tokens, cbrack())
	return tokens
}

func objectTokens(obj *hclsyntax.ObjectConsExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{obrace()}
	if len(obj.Items) > 0 {
		tokens = append(tokens, nl())
	}
	for _, item := range obj.Items {
		tokens = append(tokens, tokensForExpression(item.KeyExpr)...)
		tokens = append(tokens, assign())
		tokens = append(tokens, tokensForExpression(item.ValueExpr)...)
		tokens = append(tokens, nl())
	}
	tokens = append(tokens, cbrace())
	return tokens
}

func objectKeyTokens(key *hclsyntax.ObjectConsKeyExpr) hclwrite.Tokens {
	// TODO(i4k): review the case for key.ForceNonLiteral = true|false
	return tokensForExpression(key.Wrapped)
}

func funcallTokens(fn *hclsyntax.FunctionCallExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{ident(fn.Name, 1), oparen()}
	for i, expr := range fn.Args {
		tokens = append(tokens, tokensForExpression(expr)...)
		if i+1 != len(fn.Args) {
			tokens = append(tokens, comma())
		}
	}
	tokens = append(tokens, cparen())
	return tokens
}

func conditionalTokens(cond *hclsyntax.ConditionalExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, tokensForExpression(cond.Condition)...)
	tokens = append(tokens, question())
	tokens = append(tokens, tokensForExpression(cond.TrueResult)...)
	tokens = append(tokens, colon())
	tokens = append(tokens, tokensForExpression(cond.FalseResult)...)
	return tokens
}

func forExprTokens(forExpr *hclsyntax.ForExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	var end *hclwrite.Token
	if forExpr.KeyExpr != nil {
		// it's an object for-expr
		end = cbrace()
		tokens = append(tokens, obrace(), ident("for", 0))
		if forExpr.KeyVar != "" {
			tokens = append(tokens, ident(forExpr.KeyVar, 1))
			tokens = append(tokens, comma())
		}
		tokens = append(tokens, ident(forExpr.ValVar, 1))
	} else {
		end = cbrack()
		tokens = append(tokens, obrack(), ident("for", 0))
		tokens = append(tokens, ident(forExpr.ValVar, 1))
	}
	tokens = append(tokens, ident("in", 1))
	tokens = append(tokens, tokensForExpression(forExpr.CollExpr)...)
	tokens = append(tokens, colon())
	if forExpr.KeyExpr != nil {
		tokens = append(tokens, tokensForExpression(forExpr.KeyExpr)...)
		tokens = append(tokens, arrow())
		tokens = append(tokens, tokensForExpression(forExpr.ValExpr)...)
	} else {
		tokens = append(tokens, tokensForExpression(forExpr.ValExpr)...)
	}
	if forExpr.CondExpr != nil {
		tokens = append(tokens, ident("if", 1))
		tokens = append(tokens, tokensForExpression(forExpr.CondExpr)...)
	}
	tokens = append(tokens, end)
	return tokens
}

func indexTokens(index *hclsyntax.IndexExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, tokensForExpression(index.Collection)...)
	tokens = append(tokens, obrack())
	tokens = append(tokens, tokensForExpression(index.Key)...)
	tokens = append(tokens, cbrack())
	return tokens
}

func splatTokens(splat *hclsyntax.SplatExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, tokensForExpression(splat.Source)...)
	tokens = append(tokens, obrack())
	tokens = append(tokens, star())
	tokens = append(tokens, cbrack())
	tokens = append(tokens, tokensForExpression(splat.Each)...)

	return tokens
}

func scopeTraversalTokens(scope *hclsyntax.ScopeTraversalExpr) hclwrite.Tokens {
	return traversalTokens(scope.Traversal)
}

func traversalTokens(traversals hcl.Traversal) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	for i, traversal := range traversals {
		switch t := traversal.(type) {
		case hcl.TraverseRoot:
			if i > 0 {
				panic("malformed hcl")
			}
			tokens = append(tokens, ident(t.Name, 1))
		case hcl.TraverseAttr:
			tokens = append(tokens, dot(), ident(t.Name, 0))
		case hcl.TraverseIndex:
			tokens = append(tokens, obrack())
			tokens = append(tokens, hclwrite.TokensForValue(t.Key)...)
			tokens = append(tokens, cbrack())
		default:
			panic(fmt.Sprintf("type %T\n", t))
		}
	}
	return tokens
}

func relTraversalTokens(traversal *hclsyntax.RelativeTraversalExpr) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, tokensForExpression(traversal.Source)...)
	tokens = append(tokens, traversalTokens(traversal.Traversal)...)
	return tokens
}

func obrace() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte{'{'},
	}
}

func cbrace() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte{'}'},
	}
}

func oparen() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte{'('},
	}
}

func cparen() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte{')'},
	}
}

func obrack() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte{'['},
	}
}

func cbrack() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte{']'},
	}
}

func star() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenStar,
		Bytes: []byte{'*'},
	}
}

func interpBegin() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenTemplateInterp,
		Bytes: []byte{'$', '{'},
	}
}

func interpEnd() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenTemplateSeqEnd,
		Bytes: []byte{'}'},
	}
}

func percent() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenPercent,
		Bytes: []byte{'%'},
	}
}

func assign() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenEqual,
		Bytes: []byte{'='},
	}
}

func equal() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenEqualOp,
		Bytes: []byte{'=', '='},
	}
}

func nequal() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenNotEqual,
		Bytes: []byte{'!', '='},
	}
}

func gtr() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenGreaterThan,
		Bytes: []byte{'>'},
	}
}

func gtreq() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenGreaterThanEq,
		Bytes: []byte{'>', '='},
	}
}

func arrow() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenFatArrow,
		Bytes: []byte{'=', '>'},
	}
}

func lss() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenLessThan,
		Bytes: []byte{'<'},
	}
}

func lsseq() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenLessThanEq,
		Bytes: []byte{'<', '='},
	}
}

func bang() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenBang,
		Bytes: []byte{'!'},
	}
}

func or() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenOr,
		Bytes: []byte{'|', '|'},
	}
}

func and() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenAnd,
		Bytes: []byte{'&', '&'},
	}
}

func comma() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenComma,
		Bytes: []byte{','},
	}
}

func colon() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenColon,
		Bytes: []byte{':'},
	}
}

func question() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenQuestion,
		Bytes: []byte{'?'},
	}
}

func dot() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenDot,
		Bytes: []byte{'.'},
	}
}

func ident(name string, spaces int) *hclwrite.Token {
	return &hclwrite.Token{
		Type:         hclsyntax.TokenIdent,
		Bytes:        []byte(name),
		SpacesBefore: spaces,
	}
}

func nl() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte{'\n'},
	}
}

func add() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenPlus,
		Bytes: []byte{'+'},
	}
}

func minus() *hclwrite.Token {
	return &hclwrite.Token{
		Type:         hclsyntax.TokenMinus,
		Bytes:        []byte{'-'},
		SpacesBefore: 1,
	}
}

func slash() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenSlash,
		Bytes: []byte{'/'},
	}
}

func oquote() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenOQuote,
		Bytes: []byte{'"'},
	}
}

func cquote() *hclwrite.Token {
	return &hclwrite.Token{
		Type:  hclsyntax.TokenCQuote,
		Bytes: []byte{'"'},
	}
}

func eof() *hclwrite.Token {
	return &hclwrite.Token{
		Type: hclsyntax.TokenEOF,
	}
}

func anonSplatTokens(anon *hclsyntax.AnonSymbolExpr) hclwrite.Tokens {
	// this node is solely used during the splat evaluation.
	return hclwrite.Tokens{}
}
