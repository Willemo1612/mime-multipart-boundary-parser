# mime-multipart-boundary-parser

Parses a MIME multipart message body into its headers and content parts given a boundary string.

```go
package main

import (
	"fmt"
	"log"

	mime_multipart_boundary_parser "mime-multipart-boundary-parser"
)

func main() {
	body := []byte("--BOUNDARY\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"hello\r\n" +
		"--BOUNDARY--\r\n")

	parts, err := mime_multipart_boundary_parser.Parse(body, "BOUNDARY")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range parts {
		fmt.Printf("type=%s body=%q\n", p.HeaderValue("Content-Type"), p.Body)
	}
}
```

## Why this exists

Splitting a multipart body on a boundary looks trivial until a part body contains a line that looks like the boundary. `bytes.Split` and `strings.Index` will split there, corrupting the message. This package scans line by line and only treats a line as a delimiter when it is exactly `--` + boundary, optionally followed by `--` for the closing marker. That is the rule RFC 2046 actually specifies.

The trade-off is strictness. We do not attempt to recover from malformed input; if the opening or closing delimiter is missing, `Parse` returns an error instead of a partial result. Callers that need lenient recovery should not use this package.

## API

- `Parse(body []byte, boundary string) ([]Part, error)` — splits `body` on `boundary`. Returns one `Part` per section. Preamble before the first delimiter and epilogue after the closing delimiter are discarded.
- `Part` — has exported fields `Headers []byte` and `Body []byte`, a `HeaderValue(name string) string` method, and a `String() string` method for debugging.

Header blocks are returned with their original line endings normalised to CRLF. A part with no blank line after its headers is treated as headers-only, with an empty body.

## The awkward edge

A line in a part body that begins with the boundary text but continues with other characters is *not* a delimiter and is kept as body content. This is deliberate: it is the whole reason the library exists. If you need to embed a literal line that equals the boundary, you must choose a different boundary, exactly as MIME requires.
