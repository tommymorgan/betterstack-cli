package pattern

import "strings"

// WhereCondition builds a safe tail-query where_condition clause that matches
// a literal substring anywhere in the message field. Special characters are
// escaped so the pattern behaves as a literal match, never as regex or query
// syntax.
func WhereCondition(literal string) string {
	return "message CONTAINS " + quote(literal)
}

// quote wraps a string in double quotes, escaping backslashes and quotes
// inside the string.
func quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
