package mcp

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type sqlTokenKind int

const (
	sqlTokenIdent sqlTokenKind = iota
	sqlTokenString
	sqlTokenNumber
	sqlTokenSymbol
)

type sqlToken struct {
	kind sqlTokenKind
	text string
}

var mcpAllowedTables = map[string]bool{
	"content":    true,
	"events":     true,
	"indicators": true,
	"metrics":    true,
	"projects":   true,
}

var mcpForbiddenKeywords = map[string]bool{
	"alter":      true,
	"attach":     true,
	"call":       true,
	"checkpoint": true,
	"copy":       true,
	"create":     true,
	"delete":     true,
	"detach":     true,
	"drop":       true,
	"execute":    true,
	"export":     true,
	"force":      true,
	"import":     true,
	"insert":     true,
	"install":    true,
	"load":       true,
	"merge":      true,
	"pragma":     true,
	"reset":      true,
	"set":        true,
	"truncate":   true,
	"update":     true,
	"vacuum":     true,
}

var mcpForbiddenFunctions = map[string]bool{
	"current_setting": true,
	"duckdb_secrets":  true,
	"duckdb_settings": true,
	"getenv":          true,
	"glob":            true,
	"http_get":        true,
	"http_post":       true,
	"read_blob":       true,
	"read_csv":        true,
	"read_csv_auto":   true,
	"read_json":       true,
	"read_json_auto":  true,
	"read_ndjson":     true,
	"read_parquet":    true,
	"read_text":       true,
}

const defaultMCPQueryLimit = 1000

func prepareMCPQuery(query string, limit int) (string, error) {
	if _, err := validateMCPQuery(query); err != nil {
		return "", err
	}
	if limit <= 0 {
		limit = defaultMCPQueryLimit
	}
	return fmt.Sprintf("SELECT * FROM (%s) AS velocirepo_mcp_query LIMIT %d", query, limit), nil
}

func validateMCPQuery(query string) ([]sqlToken, error) {
	tokens, err := tokenizeSQL(query)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("query is empty")
	}
	if !tokenIdentEquals(tokens[0], "select") && !tokenIdentEquals(tokens[0], "with") {
		return nil, fmt.Errorf("MCP query must be a SELECT or WITH query")
	}
	if err := validateBalancedParens(tokens); err != nil {
		return nil, err
	}

	for i, tok := range tokens {
		if tok.kind == sqlTokenSymbol && tok.text == ";" {
			return nil, fmt.Errorf("MCP query must contain a single SELECT statement")
		}
		if tok.kind != sqlTokenIdent {
			continue
		}

		name := strings.ToLower(tok.text)
		if mcpForbiddenKeywords[name] {
			return nil, fmt.Errorf("MCP query cannot use %s", strings.ToUpper(name))
		}
		if isForbiddenMCPFunction(name) && nextTokenIsSymbol(tokens, i, "(") {
			return nil, fmt.Errorf("MCP query cannot call %s", name)
		}
	}

	ctes := collectMCPCTENames(tokens)
	if err := validateMCPTableRefs(tokens, ctes); err != nil {
		return nil, err
	}

	return tokens, nil
}

func tokenizeSQL(query string) ([]sqlToken, error) {
	var tokens []sqlToken
	for i := 0; i < len(query); {
		r, size := utf8.DecodeRuneInString(query[i:])
		if r == utf8.RuneError && size == 1 {
			return nil, fmt.Errorf("invalid UTF-8 in query")
		}
		if unicode.IsSpace(r) {
			i += size
			continue
		}

		if r == '-' && i+1 < len(query) && query[i+1] == '-' {
			i += 2
			for i < len(query) && query[i] != '\n' {
				i++
			}
			continue
		}
		if r == '/' && i+1 < len(query) && query[i+1] == '*' {
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("unterminated block comment")
			}
			i += end + 4
			continue
		}
		if r == '\'' {
			next, err := scanQuoted(query, i, '\'')
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, sqlToken{kind: sqlTokenString, text: query[i:next]})
			i = next
			continue
		}
		if r == '"' {
			next, text, err := scanQuotedIdentifier(query, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, sqlToken{kind: sqlTokenIdent, text: text})
			i = next
			continue
		}
		if isSQLIdentStart(r) {
			start := i
			i += size
			for i < len(query) {
				next, nextSize := utf8.DecodeRuneInString(query[i:])
				if !isSQLIdentPart(next) {
					break
				}
				i += nextSize
			}
			tokens = append(tokens, sqlToken{kind: sqlTokenIdent, text: query[start:i]})
			continue
		}
		if unicode.IsDigit(r) {
			start := i
			i += size
			for i < len(query) {
				next, nextSize := utf8.DecodeRuneInString(query[i:])
				if !unicode.IsDigit(next) && next != '.' {
					break
				}
				i += nextSize
			}
			tokens = append(tokens, sqlToken{kind: sqlTokenNumber, text: query[start:i]})
			continue
		}

		tokens = append(tokens, sqlToken{kind: sqlTokenSymbol, text: query[i : i+size]})
		i += size
	}
	return tokens, nil
}

func scanQuoted(query string, start int, quote byte) (int, error) {
	for i := start + 1; i < len(query); i++ {
		if query[i] != quote {
			continue
		}
		if i+1 < len(query) && query[i+1] == quote {
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("unterminated quoted value")
}

func scanQuotedIdentifier(query string, start int) (int, string, error) {
	var b strings.Builder
	for i := start + 1; i < len(query); i++ {
		if query[i] != '"' {
			b.WriteByte(query[i])
			continue
		}
		if i+1 < len(query) && query[i+1] == '"' {
			b.WriteByte('"')
			i++
			continue
		}
		return i + 1, b.String(), nil
	}
	return 0, "", fmt.Errorf("unterminated quoted identifier")
}

func isSQLIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isSQLIdentPart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func validateBalancedParens(tokens []sqlToken) error {
	depth := 0
	for _, tok := range tokens {
		if tok.kind != sqlTokenSymbol {
			continue
		}
		switch tok.text {
		case "(":
			depth++
		case ")":
			depth--
			if depth < 0 {
				return fmt.Errorf("unbalanced parentheses")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("unbalanced parentheses")
	}
	return nil
}

func isForbiddenMCPFunction(name string) bool {
	return mcpForbiddenFunctions[name] ||
		strings.HasPrefix(name, "read_") ||
		strings.HasSuffix(name, "_scan")
}

func nextTokenIsSymbol(tokens []sqlToken, i int, symbol string) bool {
	return i+1 < len(tokens) && tokens[i+1].kind == sqlTokenSymbol && tokens[i+1].text == symbol
}

func tokenIdentEquals(tok sqlToken, value string) bool {
	return tok.kind == sqlTokenIdent && strings.EqualFold(tok.text, value)
}

func collectMCPCTENames(tokens []sqlToken) map[string]bool {
	ctes := make(map[string]bool)
	if len(tokens) == 0 || !tokenIdentEquals(tokens[0], "with") {
		return ctes
	}

	i := 1
	if i < len(tokens) && tokenIdentEquals(tokens[i], "recursive") {
		i++
	}
	for i < len(tokens) {
		if tokens[i].kind != sqlTokenIdent {
			return ctes
		}
		ctes[strings.ToLower(tokens[i].text)] = true
		i++

		if i < len(tokens) && tokens[i].kind == sqlTokenSymbol && tokens[i].text == "(" {
			i = skipBalanced(tokens, i)
		}
		if i >= len(tokens) || !tokenIdentEquals(tokens[i], "as") {
			return ctes
		}
		i++
		if i >= len(tokens) || tokens[i].kind != sqlTokenSymbol || tokens[i].text != "(" {
			return ctes
		}
		i = skipBalanced(tokens, i)
		if i < len(tokens) && tokens[i].kind == sqlTokenSymbol && tokens[i].text == "," {
			i++
			continue
		}
		return ctes
	}
	return ctes
}

func skipBalanced(tokens []sqlToken, start int) int {
	depth := 0
	for i := start; i < len(tokens); i++ {
		if tokens[i].kind != sqlTokenSymbol {
			continue
		}
		switch tokens[i].text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(tokens)
}

func validateMCPTableRefs(tokens []sqlToken, ctes map[string]bool) error {
	depth := 0
	inFromList := make(map[int]bool)
	for i, tok := range tokens {
		if tok.kind == sqlTokenSymbol {
			switch tok.text {
			case "(":
				depth++
			case ")":
				delete(inFromList, depth)
				if depth > 0 {
					depth--
				}
			case ",":
				if inFromList[depth] {
					if err := validateMCPRelation(tokens, i+1, ctes); err != nil {
						return err
					}
				}
			}
			continue
		}
		if tok.kind != sqlTokenIdent {
			continue
		}

		name := strings.ToLower(tok.text)
		if isMCPClauseEnd(name) {
			inFromList[depth] = false
			continue
		}
		if name == "from" || name == "join" {
			if err := validateMCPRelation(tokens, i+1, ctes); err != nil {
				return err
			}
			inFromList[depth] = true
		}
	}
	return nil
}

func validateMCPRelation(tokens []sqlToken, start int, ctes map[string]bool) error {
	i := start
	for i < len(tokens) && tokens[i].kind == sqlTokenIdent {
		name := strings.ToLower(tokens[i].text)
		if name != "lateral" && name != "only" {
			break
		}
		i++
	}
	if i >= len(tokens) {
		return fmt.Errorf("MCP query has an incomplete table reference")
	}
	if tokens[i].kind == sqlTokenSymbol && tokens[i].text == "(" {
		return nil
	}
	if tokens[i].kind == sqlTokenString {
		return fmt.Errorf("MCP query cannot read tables from file paths")
	}
	if tokens[i].kind != sqlTokenIdent {
		return fmt.Errorf("MCP query can only read metrics, events, content, projects, indicators, or CTEs")
	}

	name := strings.ToLower(tokens[i].text)
	if i+2 < len(tokens) &&
		tokens[i+1].kind == sqlTokenSymbol && tokens[i+1].text == "." &&
		tokens[i+2].kind == sqlTokenIdent {
		schema := name
		name = strings.ToLower(tokens[i+2].text)
		if schema != "main" && schema != "temp" {
			return fmt.Errorf("MCP query cannot read from schema %q", schema)
		}
		i += 2
	}

	if i+1 < len(tokens) && tokens[i+1].kind == sqlTokenSymbol && tokens[i+1].text == "(" {
		return fmt.Errorf("MCP query cannot read from table functions")
	}
	if !mcpAllowedTables[name] && !ctes[name] {
		return fmt.Errorf("MCP query cannot read from table %q", name)
	}
	return nil
}

func isMCPClauseEnd(name string) bool {
	switch name {
	case "except", "fetch", "group", "having", "intersect", "limit", "minus", "offset", "order", "qualify", "sample", "union", "where", "window":
		return true
	default:
		return false
	}
}
