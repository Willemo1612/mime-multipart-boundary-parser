// Package mime_multipart_boundary_parser parses a MIME multipart message body
// into its constituent headers and content parts using a caller-supplied
// boundary string.
//
// The parser is deliberately strict: it follows RFC 2046's rule that a boundary
// delimiter is a line that begins with "--" + boundary and is followed by
// either CRLF (or LF), end-of-input, or "--" (the closing marker). Lines that
// merely contain the boundary text are not delimiters. This strictness is the
// whole point of the library: naive scanners that use strings.Index or
// bytes.Split on the boundary happily split on boundary-looking bytes inside
// part bodies, which is exactly the bug this package exists to avoid.
//
// The trade-off: we require the boundary to be well formed and we do not
// attempt to recover from malformed input. If the body does not contain a
// complete set of delimiters, Parse returns an error rather than a partial
// result. Callers that need lenient recovery should not use this package.
package mime_multipart_boundary_parser

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// Part is a single MIME part: its header block (without the terminating blank
// line) and its raw body. Headers is nil if the part has no header block.
type Part struct {
	Headers []byte
	Body    []byte
}

// Parse splits body on the given boundary and returns one Part per section.
//
// The preamble before the first delimiter and the epilogue after the closing
// delimiter are discarded, as RFC 2046 permits. A boundary that never appears
// produces an error; a boundary that appears but is never closed also produces
// an error, so callers can distinguish a truncated message from an empty one.
func Parse(body []byte, boundary string) ([]Part, error) {
	if boundary == "" {
		return nil, errors.New("mime-multipart-boundary-parser: boundary must not be empty")
	}

	delim := []byte("--" + boundary)
	lines := splitLines(body)

	var parts []Part
	var current [][]byte
	inPart := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if isDelimiter(line, delim) {
			if inPart {
				parts = append(parts, buildPart(current))
				current = nil
			}
			if isClosingDelimiter(line, delim) {
				inPart = false
				// Everything after the closing delimiter is epilogue.
				return parts, nil
			}
			inPart = true
			continue
		}

		if inPart {
			current = append(current, line)
		}
	}

	if inPart {
		return nil, errors.New("mime-multipart-boundary-parser: closing boundary not found")
	}
	return nil, errors.New("mime-multipart-boundary-parser: opening boundary not found")
}

// buildPart converts the collected lines of one section into a Part. The
// header block ends at the first blank line; everything after it is body. If
// there is no blank line the entire section is treated as headers, which
// matches the common case of a part with headers but no body.
func buildPart(lines [][]byte) Part {
	for i, line := range lines {
		if len(line) == 0 {
			return Part{
				Headers: joinLines(lines[:i]),
				Body:    joinLines(lines[i+1:]),
			}
		}
	}
	return Part{Headers: joinLines(lines)}
}

// splitLines splits b into lines, accepting both CRLF and bare LF. The
// trailing line terminator is stripped from each returned line, and a final
// line without a terminator is still returned. A trailing empty line produced
// by a terminating newline is dropped so that it does not create a spurious
// blank line in the final part.
func splitLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	lines := make([][]byte, 0, bytes.Count(b, []byte{'\n'})+1)
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			line := b[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(b) {
		line := b[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}

// isDelimiter reports whether line is a boundary delimiter: it must equal the
// delimiter exactly, or the delimiter followed by "--" (the closing marker).
// Trailing whitespace is not allowed; RFC 2046 says the delimiter is the
// boundary marker alone on a line, and being strict here prevents us from
// treating a body line that happens to start with the marker as a delimiter.
func isDelimiter(line, delim []byte) bool {
	if !bytes.HasPrefix(line, delim) {
		return false
	}
	rest := line[len(delim):]
	return len(rest) == 0 || bytes.Equal(rest, []byte("--"))
}

// isClosingDelimiter reports whether line is the closing delimiter, i.e. the
// delimiter followed by "--".
func isClosingDelimiter(line, delim []byte) bool {
	return bytes.Equal(line, append(append([]byte{}, delim...), '-', '-'))
}

// joinLines rejoins lines with CRLF. We normalise to CRLF because that is what
// MIME mandates for header folding and part bodies, and it keeps round-tripping
// predictable regardless of the line endings in the input.
func joinLines(lines [][]byte) []byte {
	var buf bytes.Buffer
	for i, line := range lines {
		if i > 0 {
			buf.WriteString("\r\n")
		}
		buf.Write(line)
	}
	return buf.Bytes()
}

// String renders a Part for debugging. It is not part of the parsing contract
// and is provided so callers can log a part without writing their own formatter.
func (p Part) String() string {
	return fmt.Sprintf("Part{Headers: %q, Body: %q}", p.Headers, p.Body)
}

// HeaderValue returns the value of the named header (case-insensitive) from
// the part's header block, or the empty string if the header is absent. It is a
// convenience for callers that only need one header and do not want to pull in
// net/textproto.
func (p Part) HeaderValue(name string) string {
	for _, line := range strings.Split(string(p.Headers), "\r\n") {
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(line[:colon]), name) {
			return strings.TrimSpace(line[colon+1:])
		}
	}
	return ""
}
