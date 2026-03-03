package sql

import "strings"

type TokenType string

const (
	TokenReveal TokenType = "REVEAL" // GET
	TokenBanish TokenType = "BANISH" // DELETE
	TokenPlant  TokenType = "PLANT"  // INSERT
	TokenMorph  TokenType = "MORPH"  // UPDATE
	TokenWith   TokenType = "WITH"
	TokenTo     TokenType = "TO"
	TokenPrefix TokenType = "PREFIX"

	TokenIdent    TokenType = "IDENT"
	TokenString   TokenType = "STRING"
	TokenNumber   TokenType = "NUMBER"
	TokenLParen   TokenType = "("
	TokenRParen   TokenType = ")"
	TokenComma    TokenType = ","
	TokenSemi     TokenType = ";"
	TokenAsterisk TokenType = "*"
	TokenEOF      TokenType = "EOF"
	TokenIllegal  TokenType = "ILLEGAL"
)

type Token struct {
	Type    TokenType
	Literal string
}

type Node interface {
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type StringLiteral struct {
	Token Token
	Value string
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) String() string  { return "'" + sl.Value + "'" }

type RevealStatement struct {
	Token    Token
	Keys     []string
	IsPrefix bool
	Prefix   string
}

func (rs *RevealStatement) statementNode() {}
func (rs *RevealStatement) String() string {
	if rs.IsPrefix {
		return "REVEAL PREFIX '" + rs.Prefix + "';"
	}
	if len(rs.Keys) == 1 {
		return "REVEAL '" + rs.Keys[0] + "';"
	}
	return "REVEAL (" + strings.Join(rs.Keys, ", ") + ");"
}

type BanishStatement struct {
	Token Token
	Keys  []string
}

func (bs *BanishStatement) statementNode() {}
func (bs *BanishStatement) String() string {
	if len(bs.Keys) == 1 {
		return "BANISH '" + bs.Keys[0] + "';"
	}
	return "BANISH (" + strings.Join(bs.Keys, ", ") + ");"
}

type KeyValuePair struct {
	Key   string
	Value string
}

type PlantStatement struct {
	Token Token
	Pairs []KeyValuePair
}

func (ps *PlantStatement) statementNode() {}
func (ps *PlantStatement) String() string {
	if len(ps.Pairs) == 1 {
		return "PLANT '" + ps.Pairs[0].Key + "' WITH '" + ps.Pairs[0].Value + "';"
	}
	var str strings.Builder
	str.WriteString("PLANT (\n")
	for _, p := range ps.Pairs {
		str.WriteString("  '" + p.Key + "' WITH '" + p.Value + "'\n")
	}
	str.WriteString(");")
	return str.String()
}

type MorphStatement struct {
	Token Token
	Pairs []KeyValuePair
}

func (ms *MorphStatement) statementNode() {}
func (ms *MorphStatement) String() string {
	if len(ms.Pairs) == 1 {
		return "MORPH '" + ms.Pairs[0].Key + "' TO '" + ms.Pairs[0].Value + "';"
	}
	var str strings.Builder
	str.WriteString("MORPH (\n")
	for _, p := range ms.Pairs {
		str.WriteString("  '" + p.Key + "' TO '" + p.Value + "'\n")
	}
	str.WriteString(");")
	return str.String()
}
