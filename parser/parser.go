// The definition of the Parser type and its functionality.
package parser

import (
	"fmt"
	"lisp/ast"
	"lisp/lexer"
	"lisp/token"
	"strconv"
)

// The Parser type is used to transform the Tokens provided by
// a Lexer into an AST that can be evaluated.
type Parser struct {
	lexer     *lexer.Lexer // The Lexer that provides the Tokens.
	curToken  token.Token  // The Token currently added to the AST.
	peekToken token.Token  // The next Token to be parsed, used for look-ahead.
	Errors    []string     // A collection of Errors encountered during parsing.
}

// Create a new Parser instance that uses the provided Lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer: l,
	}
	p.readToken()

	return p
}

// Transform the supplied Token list into an AST representing the program.
//
// This also populates the Errors field with parser errors encountered.
func (p *Parser) ParseProgram() *ast.Program {
	expressions := []ast.Expression{}

	p.readToken()
	for p.curToken.Type != token.EOF {
		expressions = append(expressions, p.parseExpression())
	}

	return &ast.Program{
		Expressions: expressions,
	}
}

// Parse Expressions recursively, to allow for nested expressions.
func (p *Parser) parseExpression() ast.Expression {
	switch p.curToken.Type {
	case token.NUM:
		float, err := strconv.ParseFloat(p.curToken.Literal, 64)

		if err == nil {
			tok := p.curToken
			p.readToken()
			return &ast.FloatLiteral{
				Token: tok,
				Value: float,
			}
		}

		errMsg := fmt.Sprintf("%s is invalid number", p.curToken.Literal)
		p.Errors = append(p.Errors, errMsg)
		p.readToken()
		return nil
	case token.STRING:
		string := &ast.StringLiteral{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		p.readToken()
		return string
	case token.IDENT:
		ident := &ast.Identifier{Token: p.curToken}
		p.readToken()
		return ident
	case token.LPAREN:
		return p.parseSExpression()
	case token.LBRACE:
		return p.parseDictLiteral()
	case token.QUOTE:
		return p.parseQuoteExpression()
	case token.EOF:
		return nil
	case token.ILLEGAL:
		p.Errors = append(p.Errors, p.curToken.Literal)
		p.readToken()
		return nil
	default:
		errorMessage := fmt.Sprintf("should not reach here:\n\treceived: %+v\n\tpeek: %s", p.curToken, p.peekToken)
		p.Errors = append(p.Errors, errorMessage)
		p.readToken()
		return nil
	}
}

// Parse SExpressions, which take the form:
//
//	(f a b c)
func (p *Parser) parseSExpression() ast.Expression {
	sExpression := &ast.SExpression{}

	p.readToken()

	if p.curToken.Type == token.RPAREN {
		p.readToken()
		return sExpression
	}

	sExpression.Fn = p.parseExpression()

	if p.curToken.Type == token.RPAREN {
		p.readToken()
		return sExpression
	}

	args := []ast.Expression{}

	for p.curToken.Type != token.RPAREN {
		if p.curToken.Type == token.EOF {
			p.Errors = append(
				p.Errors,
				"Reached EOF before ')'",
			)
			return sExpression
		}
		args = append(args, p.parseExpression())
	}

	p.readToken()
	sExpression.Args = args
	return sExpression
}

// Parse a dictionary literal of the form:
//
//	{ arg1 arg2 arg3 arg4 }
func (p *Parser) parseDictLiteral() ast.Expression {
	sExpression := &ast.SExpression{}
	sExpression.Fn = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "dict",
		},
	}

	p.readToken()

	if p.curToken.Type == token.RBRACE {
		p.readToken()
		return sExpression
	}

	args := []ast.Expression{}

	for p.curToken.Type != token.RBRACE {
		if p.curToken.Type == token.EOF {
			p.Errors = append(
				p.Errors,
				"Reached EOF before '}'",
			)
			return sExpression
		}
		args = append(args, p.parseExpression())
	}

	p.readToken()
	sExpression.Args = args
	return sExpression
}

// Parse an SExpression that begins with a quote.
//
// Currently this only parses lists of the form '(a b c).
// This is shorthand for (list a b c).
func (p *Parser) parseQuoteExpression() ast.Expression {
	sExpression := &ast.SExpression{}

	p.readToken()

	if p.curToken.Type != token.LPAREN {
		p.Errors = append(
			p.Errors,
			"' not followed by (",
		)
		return sExpression
	}
	p.readToken()

	sExpression.Fn = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "list",
		},
	}

	if p.curToken.Type == token.RPAREN {
		p.readToken()
		return sExpression
	}

	args := []ast.Expression{}

	for p.curToken.Type != token.RPAREN {
		if p.curToken.Type == token.EOF {
			p.Errors = append(
				p.Errors,
				"Reached EOF before ')'",
			)
			return sExpression
		}
		args = append(args, p.parseExpression())
	}

	p.readToken()
	sExpression.Args = args
	return sExpression
}

// Move to the next Token to parse.
func (p *Parser) readToken() token.Token {
	p.curToken = p.peekToken

	if p.curToken.Type != token.EOF {
		p.peekToken = p.lexer.NextToken()
	}

	return p.curToken
}
