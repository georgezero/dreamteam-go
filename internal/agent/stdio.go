package agent

import (
	"bufio"
	"bytes"
	"io"
)

// ServeLines is the stdio transport: one request per input line, one
// response per output line, in order, until the input ends. Blank lines
// are skipped. Only responses are written to w; a write error ends it.
func ServeLines(r io.Reader, w io.Writer, h *Handler) error {
	in := bufio.NewReader(r)
	out := bufio.NewWriter(w)
	for {
		line, err := in.ReadBytes('\n')
		if t := bytes.TrimSpace(line); len(t) > 0 {
			if _, werr := out.Write(append(h.Handle(t), '\n')); werr != nil {
				return werr
			}
			if werr := out.Flush(); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
