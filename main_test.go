package main

import (
	"runtime"
	"testing"
)

func TestNormaliseArgs(t *testing.T) {
	cases := []struct {
		input []string
		want  []string
	}{
		{
			input: []string{"/dir=C:\\Photos", "--ext", "jpg", "/dry-run"},
			want:  []string{"--dir=C:\\Photos", "--ext", "jpg", "--dry-run"},
		},
		{
			// Unix path with multiple slashes — must NOT be converted
			input: []string{"/home/user/photo.jpg"},
			want:  []string{"/home/user/photo.jpg"},
		},
		{
			// Already --flag style — unchanged
			input: []string{"--level", "journalist"},
			want:  []string{"--level", "journalist"},
		},
		{
			// Space-separated value form
			input: []string{"/dir", "D:\\Photos"},
			want:  []string{"--dir", "D:\\Photos"},
		},
		{
			input: []string{},
			want:  []string{},
		},
	}

	for _, c := range cases {
		got := normaliseArgs(c.input)
		if len(got) != len(c.want) {
			t.Errorf("normaliseArgs(%v): len=%d, want %d", c.input, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("normaliseArgs(%v)[%d] = %q, want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}

func TestFlagPrefix(t *testing.T) {
	// Help output uses the native flag style per platform; parsing still
	// accepts both everywhere (see normaliseArgs).
	want := "--"
	if runtime.GOOS == "windows" {
		want = "/"
	}
	if got := flagPrefix(); got != want {
		t.Errorf("flagPrefix() on %s = %q, want %q", runtime.GOOS, got, want)
	}
}
