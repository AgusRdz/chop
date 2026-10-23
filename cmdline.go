package main

import "strings"

// plainArgChars never change meaning on a Windows command line, quoted or not.
const plainArgChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./:=,+@%"

func isPlainArg(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(plainArgChars, rune(s[i])) {
			return false
		}
	}
	return true
}

// quoteWindowsArg wraps any non-plain argument in double quotes per the MSVC
// command-line rules. syscall.EscapeArg leaves args without whitespace unwrapped,
// and MSYS/Cygwin binaries (Git for Windows grep, sed, ...) glob unwrapped args
// from non-MSYS parents: `engines\"` becomes an opening quote and `a\|b(` loses
// its backslash, so a matching grep silently returns nothing.
func quoteWindowsArg(s string) string {
	if isPlainArg(s) {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			slashes++
		case '"':
			b.WriteString(strings.Repeat(`\`, slashes+1))
			slashes = 0
		default:
			slashes = 0
		}
		b.WriteByte(c)
	}
	b.WriteString(strings.Repeat(`\`, slashes))
	b.WriteByte('"')
	return b.String()
}

// windowsCmdLine returns the full command line for name and args, or "" when
// every argument is plain and Go's default quoting is safe to keep.
func windowsCmdLine(name string, args []string) string {
	needed := false
	for _, a := range args {
		if !isPlainArg(a) {
			needed = true
			break
		}
	}
	if !needed {
		return ""
	}
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteWindowsArg(name))
	for _, a := range args {
		parts = append(parts, quoteWindowsArg(a))
	}
	return strings.Join(parts, " ")
}
