package ingest

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

// ADR-161 Decision 13. The streaming reader drops exactly what TrimPipedCRLF
// drops, whatever size the reads come in, and says so once.
func TestThePipedCRLFReaderAgreesWithTrimPipedCRLF(t *testing.T) {
	for _, in := range []string{
		"", "\r\n", "a", "a\n", "a\n\r\n", "a\nb\n\r\n", "a\r\n", "a\r\nb\r\n", "a\r\nb\r\n\r\n",
		"a\nb\r\n", "a\rb\n\r\n", "\n\r\n", "x\r\n\r\n", "@@ f 1 replace\nA\n\r\n", "a\n\r\nb\n",
	} {
		want, wantCut := TrimPipedCRLF([]byte(in))
		for name, wrap := range map[string]func(io.Reader) io.Reader{
			"whole":    func(r io.Reader) io.Reader { return r },
			"one byte": iotest.OneByteReader,
			"half":     iotest.HalfReader,
		} {
			cut := 0
			p := NewPipedCRLFReader(wrap(bytes.NewReader([]byte(in))))
			p.OnTrim = func() { cut++ }
			got, err := io.ReadAll(p)
			if err != nil || !bytes.Equal(got, want) || (cut == 1) != wantCut || cut > 1 {
				t.Errorf("%q (%s reads): got %q cut %d err %v; want %q cut %v", in, name, got, cut, err, want, wantCut)
			}
		}
	}
}
