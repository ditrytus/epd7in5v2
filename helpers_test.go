package epd7in5v2

import (
	"bytes"
	"fmt"
	"testing"
)

// assertOps compares the recorded operation stream against the expected one,
// reporting the first divergence in a readable form.
func assertOps(t *testing.T, got, want []op) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("op count = %d, want %d\n got: %s\nwant: %s",
			len(got), len(want), formatOps(got), formatOps(want))
	}
	for i := range want {
		if got[i].cmd != want[i].cmd || !bytes.Equal(got[i].params, want[i].params) {
			t.Errorf("op[%d] = {%#02x, % X}, want {%#02x, % X}",
				i, got[i].cmd, got[i].params, want[i].cmd, want[i].params)
		}
	}
}

func formatOps(ops []op) string {
	var b bytes.Buffer
	b.WriteString("[")
	for i, o := range ops {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "{%#02x % X}", o.cmd, o.params)
	}
	b.WriteString("]")
	return b.String()
}

func repeatByte(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
