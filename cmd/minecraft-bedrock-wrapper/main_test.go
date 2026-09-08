package main

import "testing"

func TestRun_HelpReturnsZero(t *testing.T) {
	if got := run([]string{"--help"}); got != 0 {
		t.Fatalf("run(--help) = %d; want 0", got)
	}
}
