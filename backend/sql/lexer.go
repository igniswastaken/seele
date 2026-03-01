package sql

import "strings"

type Lexer struct {
	input        string
	position     int  // curr pos
	readPosition int  // curr reading pos in input
	ch           byte // curr char
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII Code for NUL / EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) NextToken() Token {
	var token Token

	l.skipWhitespace()

	switch l.ch {
	case '=':
		token = Token{Type: TokenIllegal, Literal: string(l.ch)}
	case ';':
		token = Token{Type: TokenSemi, Literal: string(l.ch)}
	case '(':
		token = Token{Type: TokenLParen, Literal: string(l.ch)}
	case ')':
		token = Token{Type: TokenRParen, Literal: string(l.ch)}
	case ',':
		token = Token{Type: TokenComma, Literal: string(l.ch)}
	case '\'':
		token.Type = TokenString
		token.Literal = l.readString()
	case '"':
		token.Type = TokenString
		token.Literal = l.readDoubleQuoteString()
	case 0:
		token.Literal = ""
		token.Type = TokenEOF

	default:
		if isLetter(l.ch) {
			token.Literal = l.readIdentifier()
			token.Type = lookupIdent(token.Literal)
			return token
		} else if isDigit(l.ch) {
			token.Type = TokenNumber
			token.Literal = l.readNumber()
			return token
		}
		token = Token{Type: TokenIllegal, Literal: string(l.ch)}
	}

	l.readChar()
	return token
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || l.ch == '-' || l.ch == ':' || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}
func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readDoubleQuoteString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

var keywords = map[string]TokenType{
	"REVEAL": TokenReveal,
	"BANISH": TokenBanish,
	"PLANT":  TokenPlant,
	"MORPH":  TokenMorph,
	"WITH":   TokenWith,
	"TO":     TokenTo,
	"PREFIX": TokenPrefix,
}

func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[strings.ToUpper(ident)]; ok {
		return tok
	}
	return TokenIdent
}
