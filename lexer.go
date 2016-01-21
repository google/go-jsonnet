package jsonnet

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

//////////////////////////////////////////////////////////////////////////////
// Fodder
//
// Fodder is stuff that is usually thrown away by lexers/preprocessors but is
// kept so that the source can be round tripped with full fidelity.
type fodderKind int

const (
	fodderWhitespace fodderKind = iota
	fodderCommentC
	fodderCommentCpp
	fodderCommentHash
)

type fodderElement struct {
	kind fodderKind
	data string
}

type fodder []fodderElement

//////////////////////////////////////////////////////////////////////////////
// Token

type tokenKind int

const (
	tokenInvalid tokenKind = iota

	// Symbols
	tokenBraceL
	tokenBraceR
	tokenBracketL
	tokenBracketR
	tokenColon
	tokenComma
	tokenDollar
	tokenDot
	tokenParenL
	tokenParenR
	tokenSemicolon

	// Arbitrary length lexemes
	tokenIdentifier
	tokenNumber
	tokenOperator
	tokenStringDouble
	tokenStringSingle
	tokenStringBlock

	// Keywords
	tokenAssert
	tokenElse
	tokenError
	tokenFalse
	tokenFor
	tokenFunction
	tokenIf
	tokenImport
	tokenImportStr
	tokenIn
	tokenLocal
	tokenNullLit
	tokenTailStrict
	tokenThen
	tokenSelf
	tokenSuper
	tokenTrue

	// A special token that holds line/column information about the end of the
	// file.
	tokenEndOfFile
)

type token struct {
	kind   tokenKind // The type of the token
	fodder fodder    // Any fodder the occurs before this token
	data   string    // Content of the token if it is not a keyword

	// Extra info for when kind == tokenStringBlock
	stringBlockIndent     string // The sequence of whitespace that indented the block.
	stringBlockTermIndent string // This is always fewer whitespace characters than in stringBlockIndent.

	loc LocationRange
}

type tokens []token

//////////////////////////////////////////////////////////////////////////////
// Helpers

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isNumber(r rune) bool {
	return r >= '0' && r <= '9'
}

func isIdentifierFirst(r rune) bool {
	return isUpper(r) || isLower(r) || r == '_'
}

func isIdentifier(r rune) bool {
	return isIdentifierFirst(r) || isNumber(r)
}

func isSymbol(r rune) bool {
	switch r {
	case '&', '|', '^', '=', '<', '>', '*', '/', '%', '#':
		return true
	}
	return false
}

// Check that b has at least the same whitespace prefix as a and returns the
// amount of this whitespace, otherwise returns 0.  If a has no whitespace
// prefix than return 0.
func checkWhitespace(a, b string) int {
	i := 0
	for ; i < len(a); i++ {
		if a[i] != ' ' && a[i] != '\t' {
			// a has run out of whitespace and b matched up to this point.  Return
			// result.
			return i
		}
		if i >= len(b) {
			// We ran off the edge of b while a still has whitespace.  Return 0 as
			// failure.
			return 0
		}
		if a[i] != b[i] {
			// a has whitespace but b does not.  Return 0 as failure.
			return 0
		}
	}
	// We ran off the end of a and b kept up
	return i
}

//////////////////////////////////////////////////////////////////////////////
// Lexer

type lexer struct {
	fileName string // The file name being lexed, only used for errors
	input    string // The input string

	pos        int // Current byte position in input
	lineNumber int // Current line number for pos
	lineStart  int // Byte position of start of line

	// Data about the state position of the lexer before previous call to
	// 'next'. If this state is lost then prevPos is set to lexEOF and panic
	// ensues.
	prevPos        int // Byte position of last rune read
	prevLineNumber int // The line number before last rune read
	prevLineStart  int // The line start before last rune read

	tokens tokens // The tokens that we've generated so far

	// Information about the token we are working on right now
	fodder        fodder
	tokenStart    int
	tokenStartLoc Location
}

const lexEOF = -1

func makeLexer(fn string, input string) *lexer {
	return &lexer{
		fileName:       fn,
		input:          input,
		lineNumber:     1,
		prevPos:        lexEOF,
		prevLineNumber: 1,
		tokenStartLoc:  Location{Line: 1, Column: 1},
	}
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.prevPos = l.pos
		return lexEOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.prevPos = l.pos
	l.pos += w
	if r == '\n' {
		l.prevLineNumber = l.lineNumber
		l.prevLineStart = l.lineStart
		l.lineNumber++
		l.lineStart = l.pos
	}
	return r
}

func (l *lexer) acceptN(n int) {
	for i := 0; i < n; i++ {
		l.next()
	}
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	if l.prevPos == lexEOF {
		panic("backup called with no valid previous rune")
	}
	l.lineNumber = l.prevLineNumber
	l.lineStart = l.prevLineStart
	l.pos = l.prevPos
	l.prevPos = lexEOF
}

func (l *lexer) location() Location {
	return Location{Line: l.lineNumber, Column: l.pos - l.lineStart + 1}
}

func (l *lexer) prevLocation() Location {
	if l.prevPos == lexEOF {
		panic("prevLocation called with no valid previous rune")
	}
	return Location{Line: l.prevLineNumber, Column: l.prevPos - l.prevLineStart + 1}
}

// Reset the current working token start to the current cursor position.  This
// may throw away some characters.  This does not throw away any accumulated
// fodder.
func (l *lexer) resetTokenStart() {
	l.tokenStart = l.pos
	l.tokenStartLoc = l.location()
}

func (l *lexer) emitFullToken(kind tokenKind, data, stringBlockIndent, stringBlockTermIndent string) {
	l.tokens = append(l.tokens, token{
		kind:                  kind,
		fodder:                l.fodder,
		data:                  data,
		stringBlockIndent:     stringBlockIndent,
		stringBlockTermIndent: stringBlockTermIndent,
		loc: makeLocationRange(l.fileName, l.tokenStartLoc, l.location()),
	})
	l.fodder = fodder{}
}

func (l *lexer) emitToken(kind tokenKind) {
	l.emitFullToken(kind, l.input[l.tokenStart:l.pos], "", "")
	l.resetTokenStart()
}

func (l *lexer) addWhitespaceFodder() {
	fodderData := l.input[l.tokenStart:l.pos]
	if len(l.fodder) == 0 || l.fodder[len(l.fodder)-1].kind != fodderWhitespace {
		l.fodder = append(l.fodder, fodderElement{kind: fodderWhitespace, data: fodderData})
	} else {
		l.fodder[len(l.fodder)-1].data += fodderData
	}
	l.resetTokenStart()
}

func (l *lexer) addCommentFodder(kind fodderKind) {
	fodderData := l.input[l.tokenStart:l.pos]
	l.fodder = append(l.fodder, fodderElement{kind: kind, data: fodderData})
	l.resetTokenStart()
}

func (l *lexer) addFodder(kind fodderKind, data string) {
	l.fodder = append(l.fodder, fodderElement{kind: kind, data: data})
}

// lexNumber will consume a number and emit a token.  It is assumed
// that the next rune to be served by the lexer will be a leading digit.
func (l *lexer) lexNumber() error {
	// This function should be understood with reference to the linked image:
	// http://www.json.org/number.gif

	// Note, we deviate from the json.org documentation as follows:
	// There is no reason to lex negative numbers as atomic tokens, it is better to parse them
	// as a unary operator combined with a numeric literal.  This avoids x-1 being tokenized as
	// <identifier> <number> instead of the intended <identifier> <binop> <number>.

	type numLexState int
	const (
		numBegin numLexState = iota
		numAfterZero
		numAfterOneToNine
		numAfterDot
		numAfterDigit
		numAfterE
		numAfterExpSign
		numAfterExpDigit
	)

	state := numBegin

outerLoop:
	for true {
		r := l.next()
		switch state {
		case numBegin:
			switch {
			case r == '0':
				state = numAfterZero
			case r >= '1' && r <= '9':
				state = numAfterOneToNine
			default:
				// The caller should ensure the first rune is a digit.
				panic("Couldn't lex number")
			}
		case numAfterZero:
			switch r {
			case '.':
				state = numAfterDot
			case 'e', 'E':
				state = numAfterE
			default:
				break outerLoop
			}
		case numAfterOneToNine:
			switch {
			case r == '.':
				state = numAfterDot
			case r == 'e' || r == 'E':
				state = numAfterE
			case r >= '0' && r <= '9':
				state = numAfterOneToNine
			default:
				break outerLoop
			}
		case numAfterDot:
			switch {
			case r >= '0' && r <= '9':
				state = numAfterDigit
			default:
				return makeStaticErrorPoint(
					fmt.Sprintf("Couldn't lex number, junk after decimal point: %v", strconv.QuoteRuneToASCII(r)),
					l.fileName, l.prevLocation())
			}
		case numAfterDigit:
			switch {
			case r == 'e' || r == 'E':
				state = numAfterE
			case r >= '0' && r <= '9':
				state = numAfterDigit
			default:
				break outerLoop
			}
		case numAfterE:
			switch {
			case r == '+' || r == '-':
				state = numAfterExpSign
			case r >= '0' && r <= '9':
				state = numAfterExpDigit
			default:
				return makeStaticErrorPoint(
					fmt.Sprintf("Couldn't lex number, junk after 'E': %v", strconv.QuoteRuneToASCII(r)),
					l.fileName, l.prevLocation())
			}
		case numAfterExpSign:
			if r >= '0' && r <= '9' {
				state = numAfterExpDigit
			} else {
				return makeStaticErrorPoint(
					fmt.Sprintf("Couldn't lex number, junk after exponent sign: %v", strconv.QuoteRuneToASCII(r)),
					l.fileName, l.prevLocation())
			}

		case numAfterExpDigit:
			if r >= '0' && r <= '9' {
				state = numAfterExpDigit
			} else {
				break outerLoop
			}
		}
	}

	l.backup()
	l.emitToken(tokenNumber)
	return nil
}

// lexIdentifier will consume a identifer and emit a token.  It is assumed
// that the next rune to be served by the lexer will be a leading digit.  This
// may emit a keyword or an identifier.
func (l *lexer) lexIdentifier() {
	r := l.next()
	if !isIdentifierFirst(r) {
		panic("Unexpected character in lexIdentifier")
	}
	for ; r != lexEOF; r = l.next() {
		if !isIdentifier(r) {
			break
		}
	}
	l.backup()

	switch l.input[l.tokenStart:l.pos] {
	case "assert":
		l.emitToken(tokenAssert)
	case "else":
		l.emitToken(tokenElse)
	case "error":
		l.emitToken(tokenError)
	case "false":
		l.emitToken(tokenFalse)
	case "for":
		l.emitToken(tokenFor)
	case "function":
		l.emitToken(tokenFunction)
	case "if":
		l.emitToken(tokenIf)
	case "import":
		l.emitToken(tokenImport)
	case "importstr":
		l.emitToken(tokenImportStr)
	case "in":
		l.emitToken(tokenIn)
	case "local":
		l.emitToken(tokenLocal)
	case "null":
		l.emitToken(tokenNullLit)
	case "self":
		l.emitToken(tokenSelf)
	case "super":
		l.emitToken(tokenSuper)
	case "tailstrict":
		l.emitToken(tokenTailStrict)
	case "then":
		l.emitToken(tokenThen)
	case "true":
		l.emitToken(tokenTrue)
	default:
		// Not a keyword, assume it is an identifier
		l.emitToken(tokenIdentifier)
	}
}

// lexSymbol will lex a token that starts with a symbol.  This could be a
// comment, block quote or an operator.  This function assumes that the next
// rune to be served by the lexer will be the first rune of the new token.
func (l *lexer) lexSymbol() error {
	r := l.next()

	// Single line C++ style comment
	if r == '/' && l.peek() == '/' {
		l.next()
		l.resetTokenStart() // Throw out the leading //
		for r = l.next(); r != lexEOF && r != '\n'; r = l.next() {
		}
		// Leave the '\n' in the lexer to be fodder for the next round
		l.backup()
		l.addCommentFodder(fodderCommentCpp)
		return nil
	}

	if r == '#' {
		l.resetTokenStart() // Throw out the leading #
		for r = l.next(); r != lexEOF && r != '\n'; r = l.next() {
		}
		// Leave the '\n' in the lexer to be fodder for the next round
		l.backup()
		l.addCommentFodder(fodderCommentHash)
		return nil
	}

	if r == '/' && l.peek() == '*' {
		commentStartLoc := l.tokenStartLoc
		l.next()            // consume the '*'
		l.resetTokenStart() // Throw out the leading /*
		for r = l.next(); ; r = l.next() {
			if r == lexEOF {
				return makeStaticErrorPoint("Multi-line comment has no terminating */",
					l.fileName, commentStartLoc)
			}
			if r == '*' && l.peek() == '/' {
				commentData := l.input[l.tokenStart : l.pos-1] // Don't include trailing */
				l.addFodder(fodderCommentC, commentData)
				l.next()            // Skip past '/'
				l.resetTokenStart() // Start next token at this point
				return nil
			}
		}
	}

	if r == '|' && strings.HasPrefix(l.input[l.pos:], "||\n") {
		commentStartLoc := l.tokenStartLoc
		l.acceptN(3) // Skip "||\n"
		var cb bytes.Buffer

		// Skip leading blank lines
		for r = l.next(); r == '\n'; r = l.next() {
			cb.WriteRune(r)
		}
		l.backup()
		numWhiteSpace := checkWhitespace(l.input[l.pos:], l.input[l.pos:])
		stringBlockIndent := l.input[l.pos : l.pos+numWhiteSpace]
		if numWhiteSpace == 0 {
			return makeStaticErrorPoint("Text block's first line must start with whitespace",
				l.fileName, commentStartLoc)
		}

		for {
			if numWhiteSpace <= 0 {
				panic("Unexpected value for numWhiteSpace")
			}
			l.acceptN(numWhiteSpace)
			for r = l.next(); r != '\n'; r = l.next() {
				if r == lexEOF {
					return makeStaticErrorPoint("Unexpected EOF",
						l.fileName, commentStartLoc)
				}
				cb.WriteRune(r)
			}
			cb.WriteRune('\n')

			// Skip any blank lines
			for r = l.next(); r == '\n'; r = l.next() {
				cb.WriteRune(r)
			}
			l.backup()

			// Look at the next line
			numWhiteSpace = checkWhitespace(stringBlockIndent, l.input[l.pos:])
			if numWhiteSpace == 0 {
				// End of the text block
				var stringBlockTermIndent string
				for r = l.next(); r == ' ' || r == '\t'; r = l.next() {
					stringBlockTermIndent += string(r)
				}
				l.backup()
				if !strings.HasPrefix(l.input[l.pos:], "|||") {
					return makeStaticErrorPoint("Text block not terminated with |||",
						l.fileName, commentStartLoc)
				}
				l.acceptN(3) // Skip '|||'
				l.emitFullToken(tokenStringBlock, cb.String(),
					stringBlockIndent, stringBlockTermIndent)
				l.resetTokenStart()
				return nil
			}
		}
	}

	// Assume any string of symbols is a single operator.
	for r = l.next(); isSymbol(r); r = l.next() {

	}
	l.backup()
	l.emitToken(tokenOperator)
	return nil
}

func lex(fn string, input string) (tokens, error) {
	l := makeLexer(fn, input)

	var err error

	for r := l.next(); r != lexEOF; r = l.next() {
		switch r {
		case ' ', '\t', '\r', '\n':
			l.addWhitespaceFodder()
			continue
		case '{':
			l.emitToken(tokenBraceL)
		case '}':
			l.emitToken(tokenBraceR)
		case '[':
			l.emitToken(tokenBracketL)
		case ']':
			l.emitToken(tokenBracketR)
		case ':':
			l.emitToken(tokenColon)
		case ',':
			l.emitToken(tokenComma)
		case '$':
			l.emitToken(tokenDollar)
		case '.':
			l.emitToken(tokenDot)
		case '(':
			l.emitToken(tokenParenL)
		case ')':
			l.emitToken(tokenParenR)
		case ';':
			l.emitToken(tokenSemicolon)

			// Operators
		case '!':
			if l.peek() == '=' {
				_ = l.next()
			}
			l.emitToken(tokenOperator)
		case '~', '+', '-':
			l.emitToken(tokenOperator)

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			l.backup()
			err = l.lexNumber()
			if err != nil {
				return nil, err
			}

			// String literals
		case '"':
			stringStartLoc := l.prevLocation()
			l.resetTokenStart() // Don't include the quotes in the token data
			for r = l.next(); ; r = l.next() {
				if r == lexEOF {
					return nil, makeStaticErrorPoint("Unterminated String", l.fileName, stringStartLoc)
				}
				if r == '"' {
					l.backup()
					l.emitToken(tokenStringDouble)
					_ = l.next()
					l.resetTokenStart()
					break
				}
				if r == '\\' && l.peek() != lexEOF {
					r = l.next()
				}
			}
		case '\'':
			stringStartLoc := l.prevLocation()
			l.resetTokenStart() // Don't include the quotes in the token data
			for r = l.next(); ; r = l.next() {
				if r == lexEOF {
					return nil, makeStaticErrorPoint("Unterminated String", l.fileName, stringStartLoc)
				}
				if r == '\'' {
					l.backup()
					l.emitToken(tokenStringSingle)
					r = l.next()
					l.resetTokenStart()
					break
				}
				if r == '\\' && l.peek() != lexEOF {
					r = l.next()
				}
			}
		default:
			if isIdentifierFirst(r) {
				l.backup()
				l.lexIdentifier()
			} else if isSymbol(r) {
				l.backup()
				err = l.lexSymbol()
				if err != nil {
					return nil, err
				}
			} else {
				return nil, makeStaticErrorPoint(
					fmt.Sprintf("Could not lex the character %s", strconv.QuoteRuneToASCII(r)),
					l.fileName, l.prevLocation())
			}

		}
	}

	// We are currently at the EOF.  Emit a special token to capture any
	// trailing fodder
	l.emitToken(tokenEndOfFile)
	return l.tokens, nil
}
