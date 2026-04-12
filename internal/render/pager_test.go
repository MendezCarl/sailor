package render

import (
	"bytes"
	"os"
	"testing"
)

func TestOpenPager_NoPagerFlag(t *testing.T) {
	t.Setenv("PAGER", "less")
	pw, ok := openPager(os.Stdout, true)
	if ok {
		pw.Close()
		t.Error("expected false when noPager=true")
	}
}

func TestOpenPager_NoPagerEnv(t *testing.T) {
	t.Setenv("PAGER", "")
	pw, ok := openPager(os.Stdout, false)
	if ok {
		pw.Close()
		t.Error("expected false when $PAGER is empty")
	}
}

func TestOpenPager_NonTTY(t *testing.T) {
	t.Setenv("PAGER", "less")
	var buf bytes.Buffer
	pw, ok := openPager(&buf, false)
	if ok {
		pw.Close()
		t.Error("expected false for non-TTY writer")
	}
}
