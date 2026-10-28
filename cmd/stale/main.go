package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Alarm2024/stale/internal/check"
	"github.com/Alarm2024/stale/internal/compare"
	"github.com/Alarm2024/stale/internal/measure"
	"github.com/Alarm2024/stale/internal/redact"
	"github.com/Alarm2024/stale/internal/serve"
)

// version is set at release build time: -ldflags "-X main.version=v0.1.1".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "compare":
		os.Exit(runCompare(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println("stale", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `stale — how many slots a Solana RPC trails a reference endpoint

Usage:
  stale check <rpc-url> --ref <other-rpc-url> [--max-lag-slots N] [--for DURATION]
  stale serve --upstream URL --ref <other-rpc-url> [--listen ADDR]
  stale compare [--every DURATION] [--rounds N]
  stale version

Commands:
  check   Sample getSlot (processed) on target and reference once a second
          and print a verdict. Lag in ms is slots x 400, an estimate.
  serve   Read-only JSON-RPC proxy. Every response it forwards from the
          upstream carries X-Stale-* headers from the latest background sample.
  compare Read-only side-by-side probe of STALE_A and STALE_B from the
          environment: getHealth and getSlot every --every for --rounds,
          then a summary. Output uses labels and 8-hex fingerprints only.

The reference defaults to https://api.mainnet-beta.solana.com; a target equal
to its reference is refused, since lag against yourself is always 0.

Verdicts: FRESH (0), STALE (1), UNKNOWN (2)
`)
}

func runCheck(args []string) int {
	targetURL, flagArgs := splitRPCURL(args)
	if targetURL == "" {
		fmt.Fprintln(os.Stderr, "check requires exactly one RPC URL argument")
		return 2
	}

	fs := flag.NewFlagSet("check", flag.ExitOnError)
	ref := fs.String("ref", measure.DefaultRefURL, "reference RPC URL")
	maxLag := fs.Int64("max-lag-slots", measure.DefaultMaxLag, "maximum acceptable lag in slots")
	forDur := fs.Duration("for", measure.DefaultSampleFor, "sampling window")
	_ = fs.Parse(flagArgs)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	verdict, err := check.Run(ctx, check.Options{
		TargetURL: targetURL,
		RefURL:    *ref,
		MaxLag:    *maxLag,
		For:       *forDur,
		Out:       os.Stdout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "check error: %v\n", err)
		return measure.VerdictUnknown.ExitCode()
	}
	return verdict.ExitCode()
}

func runCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	every := fs.Duration("every", time.Second, "interval between rounds")
	rounds := fs.Int("rounds", 10, "number of rounds before printing summary")
	_ = fs.Parse(args)

	aURL := os.Getenv("STALE_A")
	bURL := os.Getenv("STALE_B")
	if aURL == "" || bURL == "" {
		fmt.Fprintln(os.Stderr, "compare requires STALE_A and STALE_B environment variables")
		return 2
	}
	labelA := os.Getenv("STALE_A_LABEL")
	if labelA == "" {
		labelA = "A"
	}
	labelB := os.Getenv("STALE_B_LABEL")
	if labelB == "" {
		labelB = "B"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := compare.Run(ctx, compare.Options{
		A:      compare.Endpoint{Label: labelA, URL: aURL},
		B:      compare.Endpoint{Label: labelB, URL: bURL},
		Every:  *every,
		Rounds: *rounds,
		Out:    os.Stdout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare error: %v\n", err)
		return 2
	}
	return 0
}

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:8899", "listen address")
	upstream := fs.String("upstream", "", "upstream RPC URL (required)")
	ref := fs.String("ref", measure.DefaultRefURL, "reference RPC URL")
	maxLag := fs.Int64("max-lag-slots", measure.DefaultMaxLag, "maximum acceptable lag in slots")
	_ = fs.Parse(args)

	if *upstream == "" {
		fmt.Fprintln(os.Stderr, "serve requires --upstream URL")
		return 2
	}

	s, err := serve.New(serve.Options{
		Listen:   *listen,
		Upstream: *upstream,
		RefURL:   *ref,
		MaxLag:   *maxLag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		return 2
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("stale serve listening on %s upstream=%s\n", *listen, redact.URL(*upstream))
	if err := s.Start(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		return 2
	}
	return 0
}

// splitRPCURL pulls the RPC endpoint out before flag parsing so hostnames like
// mainnet-beta.solana.com are not mistaken for flags.
func splitRPCURL(args []string) (string, []string) {
	var target string
	var rest []string
	for _, arg := range args {
		if target == "" && (strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://")) {
			target = arg
			continue
		}
		rest = append(rest, arg)
	}
	return target, rest
}
