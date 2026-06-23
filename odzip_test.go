package odzip_test

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	odzip "github.com/odpay/go-odzip"
)

// payloads exercises empty input, sub-block data, the exact block boundary, and
// multi-block data that forces parallel workers to run.
func payloads(t *testing.T) map[string][]byte {
	t.Helper()
	rnd := rand.New(rand.NewSource(1))
	randBytes := func(n int) []byte {
		b := make([]byte, n)
		rnd.Read(b)
		return b
	}
	repeat := func(s string, n int) []byte {
		return bytes.Repeat([]byte(s), n)
	}
	const blockSize = 1 << 20 // ODZ_BLOCK_SIZE
	return map[string][]byte{
		"empty":               {},
		"tiny":                []byte("hi"),
		"text":                []byte("the quick brown fox jumps over the lazy dog"),
		"highly_compressible": repeat("abcabcabc123\n", 5000),
		"incompressible":      randBytes(300_000),
		"block_minus_one":     randBytes(blockSize - 1),
		"exact_block":         randBytes(blockSize),
		"multi_block":         randBytes(blockSize*3 + 12345),
		"mixed_multi_block":   append(repeat("hello world ", 200_000), randBytes(500_000)...),
	}
}

func TestRoundTripBytes(t *testing.T) {
	for name, want := range payloads(t) {
		t.Run(name, func(t *testing.T) {
			packed, err := odzip.CompressBytes(want, nil)
			if err != nil {
				t.Fatalf("CompressBytes: %v", err)
			}
			got, err := odzip.DecompressBytes(packed, nil)
			if err != nil {
				t.Fatalf("DecompressBytes: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("roundtrip mismatch: got %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

func TestCompressibleShrinks(t *testing.T) {
	want := bytes.Repeat([]byte("the same line over and over\n"), 10_000)
	packed, err := odzip.CompressBytes(want, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(packed) >= len(want) {
		t.Fatalf("expected compression: %d -> %d", len(want), len(packed))
	}
}

func TestThreadCountIsDeterministic(t *testing.T) {
	data := payloads(t)["multi_block"]
	single, err := odzip.CompressBytes(data, &odzip.Options{Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	many, err := odzip.CompressBytes(data, &odzip.Options{Threads: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(single, many) {
		t.Fatal("output differs by thread count; format should be thread-invariant")
	}
}

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := payloads(t)["mixed_multi_block"]

	srcPath := filepath.Join(dir, "input.bin")
	if err := os.WriteFile(srcPath, want, 0o644); err != nil {
		t.Fatal(err)
	}
	odzPath := filepath.Join(dir, "input.bin.odz")
	outPath := filepath.Join(dir, "roundtrip.bin")

	if err := odzip.CompressFile(odzPath, srcPath, nil); err != nil {
		t.Fatalf("CompressFile: %v", err)
	}
	if err := odzip.DecompressFile(outPath, odzPath, nil); err != nil {
		t.Fatalf("DecompressFile: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("file roundtrip mismatch")
	}
}

func TestStreamRoundTrip(t *testing.T) {
	want := payloads(t)["incompressible"]

	var packed bytes.Buffer
	if err := odzip.Compress(&packed, bytes.NewReader(want), nil); err != nil {
		t.Fatalf("Compress: %v", err)
	}
	var got bytes.Buffer
	if err := odzip.Decompress(&got, bytes.NewReader(packed.Bytes()), nil); err != nil {
		t.Fatalf("Decompress: %v", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatal("stream roundtrip mismatch")
	}
}

func TestProgressCallback(t *testing.T) {
	data := payloads(t)["multi_block"]
	var calls int32
	var lastProcessed, lastTotal uint64
	opts := &odzip.Options{
		Progress: func(processed, total uint64) {
			atomic.AddInt32(&calls, 1)
			lastProcessed, lastTotal = processed, total
		},
	}
	if _, err := odzip.CompressBytes(data, opts); err != nil {
		t.Fatal(err)
	}
	if calls == 0 {
		t.Fatal("progress callback was never invoked")
	}
	if lastTotal != uint64(len(data)) {
		t.Fatalf("final total = %d, want %d", lastTotal, len(data))
	}
	if lastProcessed != lastTotal {
		t.Fatalf("final processed = %d, want %d", lastProcessed, lastTotal)
	}
}

func TestDecompressGarbage(t *testing.T) {
	_, err := odzip.DecompressBytes([]byte("not an odz stream at all"), nil)
	if !errors.Is(err, odzip.ErrFormat) {
		t.Fatalf("got %v, want ErrFormat", err)
	}
}

func TestDecompressTruncated(t *testing.T) {
	packed, err := odzip.CompressBytes(payloads(t)["multi_block"], nil)
	if err != nil {
		t.Fatal(err)
	}
	// Lop off the tail so a block header/body is incomplete.
	_, err = odzip.DecompressBytes(packed[:len(packed)/2], nil)
	if err == nil {
		t.Fatal("expected an error decompressing truncated data")
	}
}

func TestDecompressFileMissing(t *testing.T) {
	err := odzip.DecompressFile(filepath.Join(t.TempDir(), "out"), filepath.Join(t.TempDir(), "nope.odz"), nil)
	if !errors.Is(err, odzip.ErrIO) {
		t.Fatalf("got %v, want ErrIO", err)
	}
}

func TestErrorMessages(t *testing.T) {
	for _, e := range []odzip.Error{odzip.ErrIO, odzip.ErrOOM, odzip.ErrFormat, odzip.ErrCorrupt} {
		if e.Error() == "" {
			t.Fatalf("empty message for code %d", int(e))
		}
	}
}
