package curl

import "strings"

// tokenize splits a shell-style string into tokens, respecting single-quoted
// strings, double-quoted strings, backslash escapes, and line continuations.
//
// Handles curl commands that span multiple lines with trailing backslashes:
//
//	curl \
//	  -X POST \
//	  https://example.com
func tokenize(input string) []string {
	var tokens []string
	var cur strings.Builder
	inToken := false

	i := 0
	for i < len(input) {
		ch := input[i]

		switch {
		case ch == '\'':
			// Single-quoted: read verbatim until the closing single quote.
			// Nothing is escaped inside single quotes.
			inToken = true
			i++ // skip opening quote
			for i < len(input) && input[i] != '\'' {
				cur.WriteByte(input[i])
				i++
			}
			if i < len(input) {
				i++ // skip closing quote
			}

		case ch == '"':
			// Double-quoted: handle backslash escapes for `"`, `\`, `$`, `` ` ``, newline.
			inToken = true
			i++ // skip opening quote
			for i < len(input) && input[i] != '"' {
				if input[i] == '\\' && i+1 < len(input) {
					next := input[i+1]
					switch next {
					case '"', '\\', '$', '`', '\n':
						cur.WriteByte(next)
					default:
						cur.WriteByte('\\')
						cur.WriteByte(next)
					}
					i += 2
				} else {
					cur.WriteByte(input[i])
					i++
				}
			}
			if i < len(input) {
				i++ // skip closing quote
			}

		case ch == '\\':
			if i+1 < len(input) {
				next := input[i+1]
				if next == '\n' {
					// Line continuation: skip both chars and treat as whitespace.
					i += 2
					// Flush current token if any — the continuation acts as a
					// token boundary in typical curl usage (the next line starts
					// a new flag), but we also want to support `--header\` with
					// the value on the next line. Standard tokenization flushes
					// here since line continuation → whitespace.
					if inToken || cur.Len() > 0 {
						tokens = append(tokens, cur.String())
						cur.Reset()
						inToken = false
					}
					continue
				}
				// Escaped character outside quotes.
				cur.WriteByte(next)
				inToken = true
				i += 2
				continue
			}
			// Trailing backslash with nothing after it: skip.
			i++

		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			if inToken || cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			i++

		default:
			cur.WriteByte(ch)
			inToken = true
			i++
		}
	}

	// Flush any remaining token.
	if inToken || cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}

	return tokens
}
