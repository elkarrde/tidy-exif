package main

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},           // shorter than limit
		{"hello world", 8, "hello..."},   // truncated with ellipsis
		{"hi", 2, "hi"},                  // exact fit
		{"hello", 3, "hel"},              // n<=3: hard cut, no ellipsis
		{"hello", 4, "h..."},             // n=4: 1 char + "..."
		{"", 10, ""},                     // empty string
		{"abcdefgh", 8, "abcdefgh"},      // exact length
		{"abcdefghi", 8, "abcde..."},     // one over
	}
	for _, c := range cases {
		got := truncate(c.input, c.n)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.input, c.n, got, c.want)
		}
		if len(got) > c.n {
			t.Errorf("truncate(%q, %d): result len %d exceeds limit", c.input, c.n, len(got))
		}
	}
}
