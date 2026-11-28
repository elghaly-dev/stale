package main

import (
	"flag"
	"strings"
	"testing"
)

func TestLeftoverArgMessageCheckRefHint(t *testing.T) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	_ = fs.Parse([]string{"ref=foo"})
	msg := leftoverArgMessage("check", fs)
	if !strings.Contains(msg, `unexpected argument "ref=foo"`) || !strings.Contains(msg, "--ref") {
		t.Fatalf("message = %q", msg)
	}
}

func TestLeftoverArgMessageUnknownNoHint(t *testing.T) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	_ = fs.Parse([]string{"nonsense"})
	msg := leftoverArgMessage("check", fs)
	if strings.Contains(msg, "did you mean") {
		t.Fatalf("unknown leftover should not suggest a flag: %q", msg)
	}
}

func TestRunCheckExits2OnLeftover(t *testing.T) {
	if code := runCheck([]string{"https://example.com/", "ref=foo"}); code != 2 {
		t.Fatalf("runCheck exit = %d, want 2", code)
	}
}

func TestRunServeExits2OnLeftover(t *testing.T) {
	if code := runServe([]string{"--upstream=https://example.com/", "bogus=1"}); code != 2 {
		t.Fatalf("runServe exit = %d, want 2", code)
	}
}

func TestLeftoverArgMessageServe(t *testing.T) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	_ = fs.String("upstream", "", "")
	_ = fs.Parse([]string{"--upstream=http://u", "max_lag_slots=3"})
	msg := leftoverArgMessage("serve", fs)
	if !strings.Contains(msg, "max_lag_slots=3") || !strings.Contains(msg, "--max-lag-slots") {
		t.Fatalf("message = %q", msg)
	}
}
