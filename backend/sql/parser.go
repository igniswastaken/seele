package sql

import "fmt"

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	errors    []string
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseStatement() Statement {
	switch p.curToken.Type {
	case TokenReveal:
		return p.parseRevealStatement()
	case TokenBanish:
		return p.parseBanishStatement()
	case TokenPlant:
		return p.parsePlantStatement()
	case TokenMorph:
		return p.parseMorphStatement()
	default:
		p.errors = append(p.errors, fmt.Sprintf("Unsupported statement starting with %s", p.curToken.Literal))
		return nil
	}
}

func (p *Parser) parseRevealStatement() *RevealStatement {
	stmt := &RevealStatement{Token: p.curToken}
	p.nextToken()

	if p.curToken.Type == TokenPrefix {
		stmt.IsPrefix = true
		p.nextToken()
		if p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
			stmt.Prefix = p.curToken.Literal
			p.nextToken()
		} else {
			p.errors = append(p.errors, "Expected string prefix after REVEAL PREFIX")
			return nil
		}
	} else if p.curToken.Type == TokenLParen {
		p.nextToken()
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			if p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
				stmt.Keys = append(stmt.Keys, p.curToken.Literal)
			} else {
				p.errors = append(p.errors, "Expected string key in REVEAL list")
			}
			p.nextToken()
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if p.curToken.Type == TokenRParen {
			p.nextToken()
		}
	} else if p.curToken.Type == TokenAsterisk {
		stmt.IsPrefix = true
		stmt.Prefix = ""
		p.nextToken()
	} else {
		if p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
			stmt.Keys = append(stmt.Keys, p.curToken.Literal)
			p.nextToken()
		} else {
			p.errors = append(p.errors, "Expected key after REVEAL")
			return nil
		}
	}

	if p.curToken.Type == TokenSemi {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseBanishStatement() *BanishStatement {
	stmt := &BanishStatement{Token: p.curToken}
	p.nextToken()

	if p.curToken.Type == TokenLParen {
		p.nextToken()
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			if p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
				stmt.Keys = append(stmt.Keys, p.curToken.Literal)
			} else {
				p.errors = append(p.errors, "Expected string key in BANISH list")
			}
			p.nextToken()
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if p.curToken.Type == TokenRParen {
			p.nextToken()
		}
	} else {
		if p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
			stmt.Keys = append(stmt.Keys, p.curToken.Literal)
			p.nextToken()
		} else {
			p.errors = append(p.errors, "Expected key after BANISH")
			return nil
		}
	}

	if p.curToken.Type == TokenSemi {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parsePlantStatement() *PlantStatement {
	stmt := &PlantStatement{Token: p.curToken}
	p.nextToken()

	if p.curToken.Type == TokenLParen {
		p.nextToken()
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			pair := p.parseKeyValuePair(TokenWith)
			if pair != nil {
				stmt.Pairs = append(stmt.Pairs, *pair)
			}
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if p.curToken.Type == TokenRParen {
			p.nextToken()
		}
	} else {
		pair := p.parseKeyValuePair(TokenWith)
		if pair != nil {
			stmt.Pairs = append(stmt.Pairs, *pair)
		}
	}

	if p.curToken.Type == TokenSemi {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseMorphStatement() *MorphStatement {
	stmt := &MorphStatement{Token: p.curToken}
	p.nextToken()

	if p.curToken.Type == TokenLParen {
		p.nextToken()
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			pair := p.parseKeyValuePair(TokenTo)
			if pair != nil {
				stmt.Pairs = append(stmt.Pairs, *pair)
			}
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if p.curToken.Type == TokenRParen {
			p.nextToken()
		}
	} else {
		pair := p.parseKeyValuePair(TokenTo)
		if pair != nil {
			stmt.Pairs = append(stmt.Pairs, *pair)
		}
	}

	if p.curToken.Type == TokenSemi {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseKeyValuePair(separator TokenType) *KeyValuePair {
	if p.curToken.Type != TokenString && p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, "Expected key")
		return nil
	}
	key := p.curToken.Literal
	p.nextToken()

	if p.curToken.Type != separator {
		p.errors = append(p.errors, fmt.Sprintf("Expected %s", separator))
		return nil
	}
	p.nextToken()

	if p.curToken.Type != TokenString && p.curToken.Type != TokenIdent && p.curToken.Type != TokenNumber {
		p.errors = append(p.errors, "Expected value")
		return nil
	}
	val := p.curToken.Literal
	p.nextToken()

	return &KeyValuePair{Key: key, Value: val}
}
