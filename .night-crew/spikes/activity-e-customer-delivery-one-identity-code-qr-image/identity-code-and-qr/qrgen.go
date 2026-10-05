// qrgen.go — THROWAWAY generator for spike E1-01. Copied by the spike script into the
// worktree's backend module as cmd/spike-qrgen/main.go so it builds against the repo's own
// vendored go-qrcode (the same library CodePNGHandler uses). Never part of the tree.
//
//   go run ./cmd/spike-qrgen <payload> <out.png> <size-px> <L|M|Q|H> <border 0|1>
//
// Prints one line: version=<n> modules=<n> bytes=<n> payload_len=<n> level=<x> border=<b>.
package main

import (
	"fmt"
	"os"
	"strconv"

	qrcode "github.com/skip2/go-qrcode"
)

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: spike-qrgen <payload> <out.png> <size-px> <L|M|Q|H> <border 0|1>")
		os.Exit(2)
	}
	payload, out := os.Args[1], os.Args[2]
	size, err := strconv.Atoi(os.Args[3])
	if err != nil || size <= 0 {
		fmt.Fprintln(os.Stderr, "bad size")
		os.Exit(2)
	}
	level := map[string]qrcode.RecoveryLevel{"L": qrcode.Low, "M": qrcode.Medium, "Q": qrcode.High, "H": qrcode.Highest}[os.Args[4]]
	border := os.Args[5] == "1"
	q, err := qrcode.New(payload, level)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	q.DisableBorder = !border
	png, err := q.PNG(size)
	if err != nil {
		fmt.Fprintln(os.Stderr, "png:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, png, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	modules := len(q.Bitmap())
	if border {
		modules -= 8 // Bitmap() includes the 4-module quiet zone each side
	}
	fmt.Printf("version=%d modules=%d bytes=%d payload_len=%d level=%s border=%v\n",
		q.VersionNumber, modules, len(png), len(payload), os.Args[4], border)
}
