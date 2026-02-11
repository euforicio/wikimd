package mermaid

import (
	"strings"
	"unicode"
)

// TokenType identifies the type of lexical token.
type TokenType int

// TokenType values.
const (
	TokenEOF TokenType = iota
	TokenNewline
	TokenWhitespace
	TokenComment
	TokenKeyword      // graph, flowchart, sequenceDiagram, etc.
	TokenDirection    // TB, BT, LR, RL, TD
	TokenIdentifier   // node IDs, participant names
	TokenString       // "text" or 'text'
	TokenLabel        // [text], (text), {text}, etc.
	TokenArrow        // -->, --->, -.->, ===>, etc.
	TokenColon        // :
	TokenSemicolon    // ;
	TokenPipe         // |
	TokenAmpersand    // &
	TokenSubgraph     // subgraph
	TokenEnd          // end
	TokenParticipant  // participant
	TokenActor        // actor
	TokenNote         // note
	TokenLoop         // loop
	TokenAlt          // alt
	TokenElse         // else
	TokenOpt          // opt
	TokenPar          // par
	TokenRect         // rect
	TokenClass        // class
	TokenNumber       // numeric values
	TokenOpenBrace    // {
	TokenCloseBrace   // }
	TokenOpenParen    // (
	TokenCloseParen   // )
	TokenOpenBracket  // [
	TokenCloseBracket // ]
	TokenPlus         // +
	TokenMinus        // -
	TokenHash         // #
	TokenTilde        // ~
	TokenDot          // .
	TokenComma        // ,
	TokenPercent      // %
	TokenLess         // <
	TokenGreater      // >
	TokenStar         // *
	TokenEquals       // =
	TokenAt           // @
)

// Token represents a lexical token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer tokenizes Mermaid syntax.
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []Token
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
	}
}

// Tokenize processes the input and returns all tokens.
func (l *Lexer) Tokenize() []Token {
	for !l.isAtEnd() {
		l.scanToken()
	}
	l.tokens = append(l.tokens, Token{Type: TokenEOF, Line: l.line, Column: l.column})
	return l.tokens
}

func (l *Lexer) scanToken() {
	ch := l.peek()

	// Skip whitespace (but track newlines)
	if ch == '\n' {
		l.addToken(TokenNewline, "\n")
		l.advance()
		l.line++
		l.column = 1
		return
	}

	if ch == '\r' {
		l.advance()
		if l.peek() == '\n' {
			l.advance()
		}
		l.addToken(TokenNewline, "\n")
		l.line++
		l.column = 1
		return
	}

	if unicode.IsSpace(rune(ch)) {
		l.skipWhitespace()
		return
	}

	// Comments
	if ch == '%' && l.peekNext() == '%' {
		l.scanComment()
		return
	}

	// Strings
	if ch == '"' || ch == '\'' {
		l.scanString(ch)
		return
	}

	// Numbers
	if unicode.IsDigit(rune(ch)) {
		l.scanNumber()
		return
	}

	// Multi-character operators (arrows, etc.)
	// Check arrows BEFORE identifiers to handle ER cardinality like "o{" correctly
	if arrow := l.tryArrow(); arrow != "" {
		l.addToken(TokenArrow, arrow)
		return
	}

	// Identifiers and keywords
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		l.scanIdentifier()
		return
	}

	// Labels with delimiters
	if label := l.tryLabel(); label != "" {
		l.addToken(TokenLabel, label)
		return
	}

	// Single character tokens
	l.scanSingleChar()
}

func (l *Lexer) scanComment() {
	start := l.pos
	l.advance() // %
	l.advance() // %

	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}
	l.addToken(TokenComment, l.input[start:l.pos])
}

func (l *Lexer) scanString(quote byte) {
	l.advance() // opening quote
	start := l.pos

	for !l.isAtEnd() && l.peek() != quote {
		if l.peek() == '\\' {
			l.advance() // escape char
		}
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}
		l.advance()
	}

	value := l.input[start:l.pos]
	if !l.isAtEnd() {
		l.advance() // closing quote
	}
	l.addToken(TokenString, value)
}

func (l *Lexer) scanNumber() {
	start := l.pos
	for !l.isAtEnd() && (unicode.IsDigit(rune(l.peek())) || l.peek() == '.') {
		l.advance()
	}
	l.addToken(TokenNumber, l.input[start:l.pos])
}

func (l *Lexer) scanIdentifier() {
	start := l.pos

scanLoop:
	for !l.isAtEnd() {
		ch := l.peek()
		// Don't include '-' in identifiers - let arrow scanner handle it
		// But allow it after alphanumeric chars for IDs like "node-1"
		switch {
		case unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_':
			l.advance()
		case ch == '-':
			// Peek ahead to check if this is an arrow
			nextCh := l.peekNext()
			if nextCh == '-' || nextCh == '>' || nextCh == '.' || nextCh == 'x' || nextCh == 'X' || nextCh == ')' {
				// This is likely an arrow, stop identifier here
				break scanLoop
			}
			// Otherwise include the dash in the identifier
			l.advance()
		default:
			break scanLoop
		}
	}

	value := l.input[start:l.pos]
	tokenType := l.identifierType(value)
	l.addToken(tokenType, value)
}

func (l *Lexer) identifierType(value string) TokenType {
	// Case-sensitive keywords (direction must be uppercase)
	caseSensitive := map[string]TokenType{
		"TB": TokenDirection,
		"TD": TokenDirection,
		"BT": TokenDirection,
		"LR": TokenDirection,
		"RL": TokenDirection,
	}

	if tt, ok := caseSensitive[value]; ok {
		return tt
	}

	// Case-insensitive keywords
	keywords := map[string]TokenType{
		"graph":              TokenKeyword,
		diagramFlowchartName: TokenKeyword,
		"sequencediagram":    TokenKeyword,
		"classdiagram":       TokenKeyword,
		"statediagram":       TokenKeyword,
		"statediagram-v2":    TokenKeyword,
		"erdiagram":          TokenKeyword,
		"gantt":              TokenKeyword,
		"pie":                TokenKeyword,
		"subgraph":           TokenSubgraph,
		"end":                TokenEnd,
		"participant":        TokenParticipant,
		"actor":              TokenActor,
		"note":               TokenNote,
		"loop":               TokenLoop,
		"alt":                TokenAlt,
		"else":               TokenElse,
		"opt":                TokenOpt,
		"par":                TokenPar,
		"rect":               TokenRect,
		"class":              TokenClass,
		"over":               TokenKeyword,
		"right":              TokenKeyword,
		"left":               TokenKeyword,
		"of":                 TokenKeyword,
		"as":                 TokenKeyword,
		"activate":           TokenKeyword,
		"deactivate":         TokenKeyword,
		"title":              TokenKeyword,
		"showdata":           TokenKeyword,
	}

	if tt, ok := keywords[strings.ToLower(value)]; ok {
		return tt
	}
	return TokenIdentifier
}

func (l *Lexer) tryArrow() string {
	// Try matching arrows from longest to shortest - ORDER MATTERS!
	// Longer patterns must come before shorter ones they contain
	arrows := []string{
		// 5+ char - include activation/deactivation markers
		"-->>+", "-->>-", "-..->", "===>", "<-->",
		// 4 char - include activation/deactivation markers
		"->>+", "->>-", "..|>", "-.->", "--->", "-->>", "-..-", "<|--", "<|..", "-->>", "-->)",
		// 3 char
		"~~~", "===", "==>", "..>", "-->", "->>", "--x", "--)", "*--", "o--", "<--", "<..", "<->", "---", "-.-",
		// 2 char
		"->", "-x", "-)", "..", "--", "}|", "|{", "}o", "o{", "||", "|o", "o|",
	}

	// ER cardinality markers that start with | - these should NOT match
	// when followed by a letter (which indicates |label| edge syntax)
	erPipeMarkers := map[string]bool{"|{": true, "||": true, "|o": true}

	for _, arrow := range arrows {
		if strings.HasPrefix(l.input[l.pos:], arrow) {
			// For ER pipe markers, check that they're not followed by a letter
			// (which would indicate a |label| edge syntax, not an ER marker)
			if erPipeMarkers[arrow] {
				nextPos := l.pos + len(arrow)
				if nextPos < len(l.input) {
					nextCh := l.input[nextPos]
					if unicode.IsLetter(rune(nextCh)) || nextCh == '_' {
						// This looks like |label|, not an ER marker
						continue
					}
				}
			}
			for range arrow {
				l.advance()
			}
			return arrow
		}
	}
	return ""
}

func (l *Lexer) tryLabel() string {
	ch := l.peek()

	// Node labels: [text], (text), {text}, etc.
	pairs := map[byte]byte{
		'[': ']',
		'(': ')',
		'{': '}',
	}

	if closer, ok := pairs[ch]; ok {
		return l.scanBracketedLabel(ch, closer)
	}

	// Special shapes: >text]
	if ch == '>' {
		return l.scanAsymmetricLabel()
	}

	return ""
}

//nolint:gocognit,gocyclo // complex label scan mirrors mermaid grammar.
func (l *Lexer) scanBracketedLabel(opener, closer byte) string {
	// Peek ahead to see if this is actually a label with content
	// or just a standalone bracket (like { at end of line for class body)
	// or empty parens like ()
	peekPos := l.pos + 1
	hasContent := false
	for peekPos < len(l.input) {
		ch := l.input[peekPos]
		if ch == '\n' || ch == '\r' {
			// Hit newline before finding content or closer - not a label
			break
		}
		if ch == closer {
			// Found closer - it's a label only if there's content
			// Empty brackets () {} [] are NOT labels
			break
		}
		if ch != ' ' && ch != '\t' {
			hasContent = true
		}
		peekPos++
	}

	// If we hit newline before closer OR there's no content, this is not a label
	if peekPos < len(l.input) && (l.input[peekPos] == '\n' || l.input[peekPos] == '\r') {
		return ""
	}
	if !hasContent {
		return "" // Empty brackets like () or {} or []
	}

	start := l.pos
	depth := 0
	l.advance() // opening bracket
	depth++

	// Handle double/triple brackets: [[text]], ((text)), {{{text}}}
	extraOpen := 0
	for l.peek() == opener {
		l.advance()
		extraOpen++
		depth++
	}

	for !l.isAtEnd() && depth > 0 {
		ch := l.peek()
		switch ch {
		case opener:
			depth++
		case closer:
			depth--
		case '\n':
			// Labels typically don't span lines - abort and reset
			l.column -= (l.pos - start)
			l.pos = start
			return ""
		}
		if depth > 0 {
			l.advance()
		}
	}

	// Consume closing brackets
	for i := 0; i <= extraOpen && !l.isAtEnd() && l.peek() == closer; i++ {
		l.advance()
	}

	return l.input[start:l.pos]
}

func (l *Lexer) scanAsymmetricLabel() string {
	start := l.pos
	l.advance() // >

	for !l.isAtEnd() && l.peek() != ']' && l.peek() != '\n' {
		l.advance()
	}

	if l.peek() == ']' {
		l.advance()
	}

	return l.input[start:l.pos]
}

//nolint:gocyclo // large switch is intentional for lexer simplicity.
func (l *Lexer) scanSingleChar() {
	ch := l.peek()
	l.advance()

	tokenType := TokenEOF
	switch ch {
	case ':':
		tokenType = TokenColon
	case ';':
		tokenType = TokenSemicolon
	case '|':
		tokenType = TokenPipe
	case '&':
		tokenType = TokenAmpersand
	case '{':
		tokenType = TokenOpenBrace
	case '}':
		tokenType = TokenCloseBrace
	case '(':
		tokenType = TokenOpenParen
	case ')':
		tokenType = TokenCloseParen
	case '[':
		tokenType = TokenOpenBracket
	case ']':
		tokenType = TokenCloseBracket
	case '+':
		tokenType = TokenPlus
	case '-':
		tokenType = TokenMinus
	case '#':
		tokenType = TokenHash
	case '~':
		tokenType = TokenTilde
	case '.':
		tokenType = TokenDot
	case ',':
		tokenType = TokenComma
	case '%':
		tokenType = TokenPercent
	case '<':
		tokenType = TokenLess
	case '>':
		tokenType = TokenGreater
	case '*':
		tokenType = TokenStar
	case '=':
		tokenType = TokenEquals
	case '@':
		tokenType = TokenAt
	}

	if tokenType != TokenEOF {
		l.addToken(tokenType, string(ch))
	}
}

func (l *Lexer) skipWhitespace() {
	for !l.isAtEnd() {
		ch := l.peek()
		if ch == ' ' || ch == '\t' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) peek() byte {
	if l.isAtEnd() {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekNext() byte {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) advance() byte {
	if l.isAtEnd() {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	l.column++
	return ch
}

func (l *Lexer) isAtEnd() bool {
	return l.pos >= len(l.input)
}

func (l *Lexer) addToken(tokenType TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:   tokenType,
		Value:  value,
		Line:   l.line,
		Column: l.column - len(value),
	})
}
