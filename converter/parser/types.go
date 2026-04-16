package parser

// IsIdentStart reports whether b can start an identifier (letter or underscore).
func IsIdentStart(b byte) bool { return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') }

// IsIdent reports whether b can appear in an identifier (letter, digit, or underscore).
func IsIdent(b byte) bool { return IsIdentStart(b) || (b >= '0' && b <= '9') }

// IsSpace reports whether b is a whitespace character.
func IsSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }
