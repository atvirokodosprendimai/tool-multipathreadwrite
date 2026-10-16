package ingest

import (
	"bytes"
	"errors"
	"io"
)

// PipedCRLFReader passes r through and, at its end, drops the CRLF PowerShell
// appends to what it pipes, by TrimPipedCRLF's rule: a final CRLF after bytes
// that hold an LF and no other CR (ADR-161 Decision 13). It holds back only the
// last two bytes, so a parser reading it streams as it would read r: a plan line
// over the parser's limit is refused before EOF, and nothing caps the whole
// plan. OnTrim, when set, is called once, at EOF, if the CRLF was dropped.
type PipedCRLFReader struct {
	r            io.Reader
	OnTrim       func()
	hold, buf    []byte
	sawLF, sawCR bool
	done         bool
	err          error
}

// NewPipedCRLFReader wraps r.
func NewPipedCRLFReader(r io.Reader) *PipedCRLFReader { return &PipedCRLFReader{r: r} }

func (p *PipedCRLFReader) Read(out []byte) (int, error) {
	for len(p.buf) == 0 {
		if p.done {
			return 0, p.err
		}
		tmp := make([]byte, max(len(out), 512))
		n, err := p.r.Read(tmp)
		data := append(p.hold, tmp[:n]...)
		p.hold = nil
		if err != nil {
			if errors.Is(err, io.EOF) {
				if body, ok := bytes.CutSuffix(data, []byte("\r\n")); ok && !p.sawCR && bytes.IndexByte(body, '\r') < 0 && (p.sawLF || bytes.IndexByte(body, '\n') >= 0) {
					data = body
					if p.OnTrim != nil {
						p.OnTrim()
					}
				}
			}
			p.buf, p.done, p.err = data, true, err
			continue
		}
		if len(data) <= 2 {
			p.hold = data
			continue
		}
		keep := len(data) - 2
		p.sawLF = p.sawLF || bytes.IndexByte(data[:keep], '\n') >= 0
		p.sawCR = p.sawCR || bytes.IndexByte(data[:keep], '\r') >= 0
		p.buf, p.hold = data[:keep], append([]byte(nil), data[keep:]...)
	}
	n := copy(out, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}
