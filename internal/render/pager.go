package render

import (
	"io"
	"os"
	"os/exec"
)

// openPager returns a WriteCloser that pipes writes through the program named
// by $PAGER. The caller must call Close() when done writing; this flushes the
// pager's stdin and waits for the pager process to exit.
//
// Returns nil, false when paging is not appropriate:
//   - noPager is true
//   - $PAGER is not set or is empty
//   - w is not a TTY (output is piped or redirected)
func openPager(w io.Writer, noPager bool) (io.WriteCloser, bool) {
	if noPager {
		return nil, false
	}
	if !isTTY(w) {
		return nil, false
	}
	pager := os.Getenv("PAGER")
	if pager == "" {
		return nil, false
	}

	cmd := exec.Command(pager)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	pw, err := cmd.StdinPipe()
	if err != nil {
		return nil, false
	}

	if err := cmd.Start(); err != nil {
		pw.Close()
		return nil, false
	}

	return &pagerWriter{pw: pw, cmd: cmd}, true
}

// pagerWriter wraps the stdin pipe of a running pager process.
// Close() flushes the pipe and waits for the pager to exit.
type pagerWriter struct {
	pw  io.WriteCloser
	cmd *exec.Cmd
}

func (p *pagerWriter) Write(b []byte) (int, error) { return p.pw.Write(b) }

func (p *pagerWriter) Close() error {
	// Close stdin so the pager knows input is complete.
	p.pw.Close()
	// Wait for the pager to exit. Ignore errors: the user quitting with 'q'
	// before all output is read causes a broken pipe, which is not an error.
	p.cmd.Wait() //nolint:errcheck
	return nil
}
