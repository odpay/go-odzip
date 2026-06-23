// Command odzip is a small command-line front-end for the go-odzip bindings.
//
// It mirrors the odz CLI's core behaviour: explicit c/d modes, or auto-detection
// from the .odz extension.
//
//	odzip [-t N] [-q] c <input> <output.odz>   compress
//	odzip [-t N] [-q] d <input.odz> <output>   decompress
//	odzip [-t N] [-q] <input> [output]         auto-detect by extension
//
// Install with:  go install github.com/odpay/go-odzip/cmd/odzip@latest
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	odzip "github.com/odpay/go-odzip"
)

func main() {
	threads := flag.Int("t", 0, "worker threads (0 = all cores)")
	quiet := flag.Bool("q", false, "suppress progress output")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	mode, in, out := parseArgs(args)
	if in == "" {
		usage()
		os.Exit(2)
	}

	opts := &odzip.Options{Threads: *threads}
	if !*quiet {
		opts.Progress = func(processed, total uint64) {
			pct := 100.0
			if total > 0 {
				pct = 100 * float64(processed) / float64(total)
			}
			fmt.Fprintf(os.Stderr, "\r  %d / %d bytes (%.1f%%)", processed, total, pct)
		}
	}

	var err error
	switch mode {
	case "c":
		err = odzip.CompressFile(out, in, opts)
	case "d":
		err = odzip.DecompressFile(out, in, opts)
	}
	if !*quiet {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "odzip: %v\n", err)
		os.Exit(1)
	}
}

// parseArgs resolves the mode (c/d), input path, and output path, auto-detecting
// from the .odz extension when no explicit mode is given.
func parseArgs(args []string) (mode, in, out string) {
	if len(args) >= 1 && (args[0] == "c" || args[0] == "d") {
		mode = args[0]
		args = args[1:]
	}
	if len(args) >= 1 {
		in = args[0]
	}
	if len(args) >= 2 {
		out = args[1]
	}
	if in == "" {
		return mode, in, out
	}
	if mode == "" {
		if strings.HasSuffix(in, ".odz") {
			mode = "d"
		} else {
			mode = "c"
		}
	}
	if out == "" {
		if mode == "c" {
			out = in + ".odz"
		} else {
			out = strings.TrimSuffix(in, ".odz")
			if out == in {
				out = in + ".out"
			}
		}
	}
	return mode, in, out
}

func usage() {
	fmt.Fprint(os.Stderr, `odzip — LZ77+Huffman compressor (go-odzip bindings)

usage:
  odzip [options] c <input> <output.odz>   compress
  odzip [options] d <input.odz> <output>   decompress
  odzip [options] <input> [output]         auto-detect by extension

options:
  -t N   worker threads (0 = all cores)
  -q     suppress progress output
`)
}
