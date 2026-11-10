package mime_multipart_boundary_parser

import (
	"strings"
	"testing"
)

func TestParseSimple(t *testing.T) {
	body := "--BOUNDARY\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"hello\r\n" +
		"--BOUNDARY\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"world\r\n" +
		"--BOUNDARY--\r\n"

	parts, err := Parse([]byte(body), "BOUNDARY")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	if got := parts[0].HeaderValue("Content-Type"); got != "text/plain" {
		t.Errorf("part 0 Content-Type = %q, want text/plain", got)
	}
	if got := string(parts[0].Body); got != "hello" {
		t.Errorf("part 0 body = %q, want hello", got)
	}
	if got := string(parts[1].Body); got != "world" {
		t.Errorf("part 1 body = %q, want world", got)
	}
}

func TestParseBareLF(t *testing.T) {
	body := "--B\n" +
		"Content-Type: text/plain\n" +
		"\n" +
		"line1\n" +
		"line2\n" +
		"--B--\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if got := string(parts[0].Body); got != "line1\r\nline2" {
		t.Errorf("body = %q, want line1\\r\\nline2", got)
	}
}

func TestParseNoHeaders(t *testing.T) {
	body := "--B\r\n" +
		"\r\n" +
		"just a body\r\n" +
		"--B--\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if len(parts[0].Headers) != 0 {
		t.Errorf("headers = %q, want empty", parts[0].Headers)
	}
	if got := string(parts[0].Body); got != "just a body" {
		t.Errorf("body = %q, want just a body", got)
	}
}

func TestParseHeadersOnly(t *testing.T) {
	body := "--B\r\n" +
		"Content-Type: text/plain\r\n" +
		"--B--\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if got := parts[0].HeaderValue("Content-Type"); got != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
	if len(parts[0].Body) != 0 {
		t.Errorf("body = %q, want empty", parts[0].Body)
	}
}

func TestParseEmptyBody(t *testing.T) {
	body := "--B\r\n" +
		"\r\n" +
		"--B--\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if len(parts[0].Body) != 0 {
		t.Errorf("body = %q, want empty", parts[0].Body)
	}
}

func TestParsePreambleAndEpilogueDiscarded(t *testing.T) {
	body := "preamble text\r\n" +
		"--B\r\n" +
		"\r\n" +
		"body\r\n" +
		"--B--\r\n" +
		"epilogue text\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if got := string(parts[0].Body); got != "body" {
		t.Errorf("body = %q, want body", got)
	}
}

func TestParseBoundaryLikeLineInBody(t *testing.T) {
	// A line that contains the boundary text but is not a delimiter must not
	// split the part. This is the bug the package exists to prevent.
	body := "--B\r\n" +
		"\r\n" +
		"this line mentions --B but is not a delimiter\r\n" +
		"--B--\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	want := "this line mentions --B but is not a delimiter"
	if got := string(parts[0].Body); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestParseMissingClosingBoundary(t *testing.T) {
	body := "--B\r\n" +
		"\r\n" +
		"body\r\n"

	_, err := Parse([]byte(body), "B")
	if err == nil {
		t.Fatal("Parse returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "closing boundary not found") {
		t.Errorf("error = %v, want closing boundary not found", err)
	}
}

func TestParseMissingOpeningBoundary(t *testing.T) {
	body := "no boundary here\r\n"

	_, err := Parse([]byte(body), "B")
	if err == nil {
		t.Fatal("Parse returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "opening boundary not found") {
		t.Errorf("error = %v, want opening boundary not found", err)
	}
}

func TestParseEmptyBoundary(t *testing.T) {
	_, err := Parse([]byte("anything"), "")
	if err == nil {
		t.Fatal("Parse returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "boundary must not be empty") {
		t.Errorf("error = %v, want boundary must not be empty", err)
	}
}

func TestParseEmptyBodyInput(t *testing.T) {
	_, err := Parse(nil, "B")
	if err == nil {
		t.Fatal("Parse returned nil error, want error")
	}
}

func TestParseNoTrailingNewline(t *testing.T) {
	body := "--B\r\n" +
		"\r\n" +
		"body\r\n" +
		"--B--"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if got := string(parts[0].Body); got != "body" {
		t.Errorf("body = %q, want body", got)
	}
}

func TestPartHeaderValueCaseInsensitive(t *testing.T) {
	body := "--B\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"x\r\n" +
		"--B--\r\n"

	parts, err := Parse([]byte(body), "B")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got := parts[0].HeaderValue("content-type"); got != "text/plain" {
		t.Errorf("content-type = %q, want text/plain", got)
	}
	if got := parts[0].HeaderValue("X-Missing"); got != "" {
		t.Errorf("X-Missing = %q, want empty", got)
	}
}

func TestPartString(t *testing.T) {
	p := Part{Headers: []byte("A: b"), Body: []byte("c")}
	got := p.String()
	if !strings.Contains(got, "Headers") || !strings.Contains(got, "Body") {
		t.Errorf("String() = %q, want it to mention Headers and Body", got)
	}
}
