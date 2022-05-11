/*
Copyright 2016 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package parser

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
)

var (
	tEOF = token{kind: tokenEndOfFile}
)

func fodderEqual(f1 ast.Fodder, f2 ast.Fodder) bool {
	if len(f1) != len(f2) {
		return false
	}
	for i := range f1 {
		if f1[i].Kind != f2[i].Kind {
			return false
		}
		if f1[i].Blanks != f2[i].Blanks {
			return false
		}
		if f1[i].Indent != f2[i].Indent {
			return false
		}
		if len(f1[i].Comment) != len(f2[i].Comment) {
			return false
		}
		for j := range f1[i].Comment {
			if f1[i].Comment[j] != f2[i].Comment[j] {
				return false
			}
		}
	}
	return true
}

func tokensEqual(ts1, ts2 Tokens) bool {
	if len(ts1) != len(ts2) {
		return false
	}
	for i := range ts1 {
		t1, t2 := ts1[i], ts2[i]
		if t1.kind != t2.kind {
			return false
		}
		if t1.data != t2.data {
			return false
		}
		if !fodderEqual(t1.fodder, t2.fodder) {
			return false
		}
		if t1.stringBlockIndent != t2.stringBlockIndent {
			return false
		}
		if t1.stringBlockTermIndent != t2.stringBlockTermIndent {
			return false
		}
	}
	return true
}

func SingleTest(t *testing.T, input string, expectedError string, expected Tokens) {
	// Copy the test tokens and append an EOF token
	testTokens := append(Tokens(nil), expected...)
	if len(testTokens) == 0 || testTokens[len(testTokens)-1].kind != tokenEndOfFile {
		testTokens = append(testTokens, tEOF)
	}
	tokens, err := Lex("snippet", "", input)
	var errString string
	if err != nil {
		errString = err.Error()
	}
	if errString != expectedError {
		t.Errorf("error result does not match. got\n\t%+v\nexpected\n\t%+v",
			errString, expectedError)
	}
	if err == nil && !tokensEqual(tokens, testTokens) {
		t.Errorf("got\n\t%+v\nexpected\n\t%+v", tokens, expected)
	}
}

// TODO: test position reporting

func TestEmpty(t *testing.T) {
	SingleTest(t, "", "", Tokens{})
}

func TestWhitespace(t *testing.T) {
	SingleTest(t, "  \t\n\r\r\n", "", Tokens{})
}

func TestBraceL(t *testing.T) {
	SingleTest(t, "{", "", Tokens{
		{kind: tokenBraceL, data: "{"},
	})
}

func TestBraceR(t *testing.T) {
	SingleTest(t, "}", "", Tokens{
		{kind: tokenBraceR, data: "}"},
	})
}

func TestBracketL(t *testing.T) {
	SingleTest(t, "[", "", Tokens{
		{kind: tokenBracketL, data: "["},
	})
}

func TestBracketR(t *testing.T) {
	SingleTest(t, "]", "", Tokens{
		{kind: tokenBracketR, data: "]"},
	})
}

func TestColon(t *testing.T) {
	SingleTest(t, ":", "", Tokens{
		{kind: tokenOperator, data: ":"},
	})
}

func TestColon2(t *testing.T) {
	SingleTest(t, "::", "", Tokens{
		{kind: tokenOperator, data: "::"},
	})
}

func TestColon3(t *testing.T) {
	SingleTest(t, ":::", "", Tokens{
		{kind: tokenOperator, data: ":::"},
	})
}

func TestArrowright(t *testing.T) {
	SingleTest(t, "->", "", Tokens{
		{kind: tokenOperator, data: "->"},
	})
}

func TestLessthanminus(t *testing.T) {
	SingleTest(t, "<-", "", Tokens{
		{kind: tokenOperator, data: "<"},
		{kind: tokenOperator, data: "-"},
	})
}

func TestComma(t *testing.T) {
	SingleTest(t, ",", "", Tokens{
		{kind: tokenComma, data: ","},
	})
}

func TestDollar(t *testing.T) {
	SingleTest(t, "$", "", Tokens{
		{kind: tokenDollar, data: "$"},
	})
}

func TestDot(t *testing.T) {
	SingleTest(t, ".", "", Tokens{
		{kind: tokenDot, data: "."},
	})
}

func TestParenL(t *testing.T) {
	SingleTest(t, "(", "", Tokens{
		{kind: tokenParenL, data: "("},
	})
}

func TestParenR(t *testing.T) {
	SingleTest(t, ")", "", Tokens{
		{kind: tokenParenR, data: ")"},
	})
}

func TestSemicolon(t *testing.T) {
	SingleTest(t, ";", "", Tokens{
		{kind: tokenSemicolon, data: ";"},
	})
}

func TestNot1(t *testing.T) {
	SingleTest(t, "!", "", Tokens{
		{kind: tokenOperator, data: "!"},
	})
}

func TestNot2(t *testing.T) {
	SingleTest(t, "! ", "", Tokens{
		{kind: tokenOperator, data: "!"},
	})
}

func TestNotequal(t *testing.T) {
	SingleTest(t, "!=", "", Tokens{
		{kind: tokenOperator, data: "!="},
	})
}

func TestTilde(t *testing.T) {
	SingleTest(t, "~", "", Tokens{
		{kind: tokenOperator, data: "~"},
	})
}

func TestPlus(t *testing.T) {
	SingleTest(t, "+", "", Tokens{
		{kind: tokenOperator, data: "+"},
	})
}

func TestMinus(t *testing.T) {
	SingleTest(t, "-", "", Tokens{
		{kind: tokenOperator, data: "-"},
	})
}

func TestNumber0(t *testing.T) {
	SingleTest(t, "0", "", Tokens{
		{kind: tokenNumber, data: "0"},
	})
}

func TestNumber1(t *testing.T) {
	SingleTest(t, "1", "", Tokens{
		{kind: tokenNumber, data: "1"},
	})
}

func TestNumber1_0(t *testing.T) {
	SingleTest(t, "1.0", "", Tokens{
		{kind: tokenNumber, data: "1.0"},
	})
}

func TestNumber0_10(t *testing.T) {
	SingleTest(t, "0.10", "", Tokens{
		{kind: tokenNumber, data: "0.10"},
	})
}

func TestNumber0e100(t *testing.T) {
	SingleTest(t, "0e100", "", Tokens{
		{kind: tokenNumber, data: "0e100"},
	})
}

func TestNumber1e100(t *testing.T) {
	SingleTest(t, "1e100", "", Tokens{
		{kind: tokenNumber, data: "1e100"},
	})
}

func TestNumber1_1e100(t *testing.T) {
	SingleTest(t, "1.1e100", "", Tokens{
		{kind: tokenNumber, data: "1.1e100"},
	})
}

func TestNumber1_1e_100(t *testing.T) {
	SingleTest(t, "1.1e-100", "", Tokens{
		{kind: tokenNumber, data: "1.1e-100"},
	})
}

func TestNumber1_1ep100(t *testing.T) {
	SingleTest(t, "1.1e+100", "", Tokens{
		{kind: tokenNumber, data: "1.1e+100"},
	})
}

func TestNumber0100(t *testing.T) {
	SingleTest(t, "0100", "", Tokens{
		{kind: tokenNumber, data: "0"},
		{kind: tokenNumber, data: "100"},
	})
}

func TestNumber10p10(t *testing.T) {
	SingleTest(t, "10+10", "", Tokens{
		{kind: tokenNumber, data: "10"},
		{kind: tokenOperator, data: "+"},
		{kind: tokenNumber, data: "10"},
	})
}

func TestNumber1_p3(t *testing.T) {
	SingleTest(t, "1.+3", "snippet:1:3 Couldn't lex number, junk after decimal point: '+'", Tokens{})
}

func TestNumber1eExc(t *testing.T) {
	SingleTest(t, "1e!", "snippet:1:3 Couldn't lex number, junk after 'E': '!'", Tokens{})
}

func TestNumber1epExc(t *testing.T) {
	SingleTest(t, "1e+!", "snippet:1:4 Couldn't lex number, junk after exponent sign: '!'", Tokens{})
}

func TestDoublestring1(t *testing.T) {
	SingleTest(t, "\"hi\"", "", Tokens{
		{kind: tokenStringDouble, data: "hi"},
	})
}

func TestDoublestring2(t *testing.T) {
	SingleTest(t, "\"hi\n\"", "", Tokens{
		{kind: tokenStringDouble, data: "hi\n"},
	})
}

func TestDoublestring3(t *testing.T) {
	SingleTest(t, "\"hi\\\"\"", "", Tokens{
		{kind: tokenStringDouble, data: "hi\\\""},
	})
}

func TestDoublestring4(t *testing.T) {
	SingleTest(t, "\"hi\\\n\"", "", Tokens{
		{kind: tokenStringDouble, data: "hi\\\n"},
	})
}

func TestDoublestring5(t *testing.T) {
	SingleTest(t, "\"hi", "snippet:1:1 Unterminated String", Tokens{})
}

func TestSinglestring1(t *testing.T) {
	SingleTest(t, "'hi'", "", Tokens{
		{kind: tokenStringSingle, data: "hi"},
	})
}

func TestSinglestring2(t *testing.T) {
	SingleTest(t, "'hi\n'", "", Tokens{
		{kind: tokenStringSingle, data: "hi\n"},
	})
}

func TestSinglestring3(t *testing.T) {
	SingleTest(t, "'hi\\''", "", Tokens{
		{kind: tokenStringSingle, data: "hi\\'"},
	})
}

func TestSinglestring4(t *testing.T) {
	SingleTest(t, "'hi\\\n'", "", Tokens{
		{kind: tokenStringSingle, data: "hi\\\n"},
	})
}

func TestSinglestring5(t *testing.T) {
	SingleTest(t, "'hi", "snippet:1:1 Unterminated String", Tokens{})
}

func TestAssert(t *testing.T) {
	SingleTest(t, "assert", "", Tokens{
		{kind: tokenAssert, data: "assert"},
	})
}

func TestElse(t *testing.T) {
	SingleTest(t, "else", "", Tokens{
		{kind: tokenElse, data: "else"},
	})
}

func TestError(t *testing.T) {
	SingleTest(t, "error", "", Tokens{
		{kind: tokenError, data: "error"},
	})
}

func TestFalse(t *testing.T) {
	SingleTest(t, "false", "", Tokens{
		{kind: tokenFalse, data: "false"},
	})
}

func TestFor(t *testing.T) {
	SingleTest(t, "for", "", Tokens{
		{kind: tokenFor, data: "for"},
	})
}

func TestFunction(t *testing.T) {
	SingleTest(t, "function", "", Tokens{
		{kind: tokenFunction, data: "function"},
	})
}

func TestIf(t *testing.T) {
	SingleTest(t, "if", "", Tokens{
		{kind: tokenIf, data: "if"},
	})
}

func TestImport(t *testing.T) {
	SingleTest(t, "import", "", Tokens{
		{kind: tokenImport, data: "import"},
	})
}

func TestImportstr(t *testing.T) {
	SingleTest(t, "importstr", "", Tokens{
		{kind: tokenImportStr, data: "importstr"},
	})
}

func TestImportbin(t *testing.T) {
	SingleTest(t, "importbin", "", Tokens{
		{kind: tokenImportBin, data: "importbin"},
	})
}

func TestIn(t *testing.T) {
	SingleTest(t, "in", "", Tokens{
		{kind: tokenIn, data: "in"},
	})
}

func TestLocal(t *testing.T) {
	SingleTest(t, "local", "", Tokens{
		{kind: tokenLocal, data: "local"},
	})
}

func TestNull(t *testing.T) {
	SingleTest(t, "null", "", Tokens{
		{kind: tokenNullLit, data: "null"},
	})
}

func TestSelf(t *testing.T) {
	SingleTest(t, "self", "", Tokens{
		{kind: tokenSelf, data: "self"},
	})
}

func TestSuper(t *testing.T) {
	SingleTest(t, "super", "", Tokens{
		{kind: tokenSuper, data: "super"},
	})
}

func TestTailstrict(t *testing.T) {
	SingleTest(t, "tailstrict", "", Tokens{
		{kind: tokenTailStrict, data: "tailstrict"},
	})
}

func TestThen(t *testing.T) {
	SingleTest(t, "then", "", Tokens{
		{kind: tokenThen, data: "then"},
	})
}

func TestTrue(t *testing.T) {
	SingleTest(t, "true", "", Tokens{
		{kind: tokenTrue, data: "true"},
	})
}

func TestIdentifier(t *testing.T) {
	SingleTest(t, "foobar123", "", Tokens{
		{kind: tokenIdentifier, data: "foobar123"},
	})
}

func TestIdentifiers(t *testing.T) {
	SingleTest(t, "foo bar123", "", Tokens{
		{kind: tokenIdentifier, data: "foo"},
		{kind: tokenIdentifier, data: "bar123"},
	})
}

func TestCppComment(t *testing.T) {
	SingleTest(t, "// hi", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderParagraph, Comment: []string{"// hi"}}}},
	})
}

func TestHashComment(t *testing.T) {
	SingleTest(t, "# hi", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderParagraph, Comment: []string{"# hi"}}}},
	})
}

func TestCComment(t *testing.T) {
	SingleTest(t, "/* hi */", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderInterstitial, Comment: []string{"/* hi */"}}}},
	})
}

func TestCCommentTooShort(t *testing.T) {
	SingleTest(t, "/*/", "snippet:1:1 Multi-line comment has no terminating */", Tokens{})
}

func TestCCommentMinimal(t *testing.T) {
	SingleTest(t, "/**/", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderInterstitial, Comment: []string{"/**/"}}}},
	})
}

func TestCCommentJustSlash(t *testing.T) {
	SingleTest(t, "/*/*/", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderInterstitial, Comment: []string{"/*/*/"}}}},
	})
}
func TestCCommentSpaceSlash(t *testing.T) {
	SingleTest(t, "/* /*/", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{{Kind: ast.FodderInterstitial, Comment: []string{"/* /*/"}}}},
	})
}

func TestCCommentManyLines(t *testing.T) {
	SingleTest(t, "/*\n\n*/", "", Tokens{
		{kind: tokenEndOfFile, fodder: ast.Fodder{
			{Kind: ast.FodderLineEnd},
			{Kind: ast.FodderParagraph, Comment: []string{"/*", "", "*/"}}}},
	})
}

func TestCCommentNoTerm(t *testing.T) {
	SingleTest(t, "/* hi", "snippet:1:1 Multi-line comment has no terminating */", Tokens{})
}

func TestBlockStringSpaces(t *testing.T) {
	SingleTest(t, "|||\n  test\n    more\n  |||\n    foo\n|||", "", Tokens{
		{
			kind:                  tokenStringBlock,
			data:                  "test\n  more\n|||\n  foo\n",
			stringBlockIndent:     "  ",
			stringBlockTermIndent: "",
		},
	})
}

func TestBlockStringTabs(t *testing.T) {
	SingleTest(t, "|||\n\ttest\n\t  more\n\t|||\n\t  foo\n|||", "", Tokens{
		{
			kind:                  tokenStringBlock,
			data:                  "test\n  more\n|||\n  foo\n",
			stringBlockIndent:     "\t",
			stringBlockTermIndent: "",
		},
	})
}

func TestBlockStringMixed(t *testing.T) {
	SingleTest(t, "|||\n\t  \ttest\n\t  \t  more\n\t  \t|||\n\t  \t  foo\n|||", "", Tokens{
		{
			kind:                  tokenStringBlock,
			data:                  "test\n  more\n|||\n  foo\n",
			stringBlockIndent:     "\t  \t",
			stringBlockTermIndent: "",
		},
	})
}

func TestBlockStringBlanks(t *testing.T) {
	SingleTest(t, "|||\n\n  test\n\n\n    more\n  |||\n    foo\n|||", "", Tokens{
		{
			kind:                  tokenStringBlock,
			data:                  "\ntest\n\n\n  more\n|||\n  foo\n",
			stringBlockIndent:     "  ",
			stringBlockTermIndent: "",
		},
	})
}

func TestBlockStringBadIndent(t *testing.T) {
	SingleTest(t, "|||\n  test\n foo\n|||", "snippet:1:1 Text block not terminated with |||", Tokens{})
}

func TestBlockStringEof(t *testing.T) {
	SingleTest(t, "|||\n  test", "snippet:1:1 Unexpected EOF", Tokens{})
}

func TestBlockStringNotTerm(t *testing.T) {
	SingleTest(t, "|||\n  test\n", "snippet:1:1 Text block not terminated with |||", Tokens{})
}

func TestBlockstringNoWs(t *testing.T) {
	SingleTest(t, "|||\ntest\n|||", "snippet:1:1 Text block's first line must start with whitespace", Tokens{})
}

func TestVerbatimString1(t *testing.T) {
	SingleTest(t, "@\"\"", "", Tokens{
		{kind: tokenVerbatimStringDouble, data: ""},
	})
}

func TestVerbatimString2(t *testing.T) {
	SingleTest(t, "@''", "", Tokens{
		{kind: tokenVerbatimStringSingle, data: ""},
	})
}

func TestVerbatimString3(t *testing.T) {
	SingleTest(t, "@\"\"\"\"", "", Tokens{
		{kind: tokenVerbatimStringDouble, data: "\""},
	})
}

func TestVerbatimString4(t *testing.T) {
	SingleTest(t, "@''''", "", Tokens{
		{kind: tokenVerbatimStringSingle, data: "'"},
	})
}

func TestVerbatimString5(t *testing.T) {
	SingleTest(t, "@\"\\n\"", "", Tokens{
		{kind: tokenVerbatimStringDouble, data: "\\n"},
	})
}

func TestVerbatimString6(t *testing.T) {
	SingleTest(t, "@\"''\"", "", Tokens{
		{kind: tokenVerbatimStringDouble, data: "''"},
	})
}

func TestVerbatimStringUnterminated(t *testing.T) {
	SingleTest(t, "@\"blah blah", "snippet:1:1 Unterminated String", Tokens{})
}

func TestVerbatimStringJunk(t *testing.T) {
	SingleTest(t, "@blah blah", "snippet:1:1 Couldn't lex verbatim string, junk after '@': 98", Tokens{})
}

func TestOpStar(t *testing.T) {
	SingleTest(t, "*", "", Tokens{
		{kind: tokenOperator, data: "*"},
	})
}

func TestOpSlash(t *testing.T) {
	SingleTest(t, "/", "", Tokens{
		{kind: tokenOperator, data: "/"},
	})
}

func TestOpPercent(t *testing.T) {
	SingleTest(t, "%", "", Tokens{
		{kind: tokenOperator, data: "%"},
	})
}

func TestOpAmp(t *testing.T) {
	SingleTest(t, "&", "", Tokens{
		{kind: tokenOperator, data: "&"},
	})
}

func TestOpPipe(t *testing.T) {
	SingleTest(t, "|", "", Tokens{
		{kind: tokenOperator, data: "|"},
	})
}

func TestOpCaret(t *testing.T) {
	SingleTest(t, "^", "", Tokens{
		{kind: tokenOperator, data: "^"},
	})
}

func TestOpEqual(t *testing.T) {
	SingleTest(t, "=", "", Tokens{
		{kind: tokenOperator, data: "="},
	})
}

func TestOpLessThan(t *testing.T) {
	SingleTest(t, "<", "", Tokens{
		{kind: tokenOperator, data: "<"},
	})
}

func TestOpGreaterThan(t *testing.T) {
	SingleTest(t, ">", "", Tokens{
		{kind: tokenOperator, data: ">"},
	})
}

func TestOpJunk(t *testing.T) {
	SingleTest(t, ">==|", "", Tokens{
		{kind: tokenOperator, data: ">==|"},
	})
}

func TestJunk(t *testing.T) {
	SingleTest(t, "💩", "snippet:1:1 Could not lex the character '\\U0001f4a9'", Tokens{})
}
