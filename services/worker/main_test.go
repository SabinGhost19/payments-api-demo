package main

import (
	"testing"
	"time"
)

func TestParseInterval(t *testing.T) {
	def := 2 * time.Second
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"5s", 5 * time.Second},
		{"500ms", 500 * time.Millisecond},
		{"", def},        // empty -> default
		{"garbage", def}, // invalid -> default
		{"0s", def},      // non-positive -> default
		{"-3s", def},     // negative -> default
	}
	for _, c := range cases {
		if got := parseInterval(c.raw, def); got != c.want {
			t.Errorf("parseInterval(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}
