// Package odzip provides Go bindings for libodzip, the LZ77+Huffman
// compressor behind the odz CLI (https://github.com/odpay/odzip).
//
// The .odz format is a DEFLATE-class container: a small header followed by
// independent 1 MB blocks, each stored raw or Huffman-coded. Blocks are
// compressed and decompressed in parallel across worker threads; the byte
// output is identical regardless of the thread count.
//
// # Importing
//
// The package vendors libodzip's C sources and builds them with cgo, so a C
// toolchain (cc/clang/gcc) is required, but no system library needs to be
// installed:
//
//	go get github.com/odpay/go-odzip
//
// # Usage
//
// The file-based functions are the most efficient and stream in bounded
// memory:
//
//	err := odzip.CompressFile("data.txt.odz", "data.txt", nil)
//	err  = odzip.DecompressFile("data.txt", "data.txt.odz", nil)
//
// Convenience helpers operate on io.Reader/io.Writer and []byte:
//
//	packed, err := odzip.CompressBytes([]byte("hello hello hello"), nil)
//	plain,  err := odzip.DecompressBytes(packed, nil)
//
// All functions take destination before source, matching io.Copy. Options
// controls threading and progress reporting; a nil *Options selects sensible
// defaults (all cores, no progress callback).
//
// # Errors
//
// Failures are reported as values of type Error. Match the sentinels with
// errors.Is:
//
//	if errors.Is(err, odzip.ErrCorrupt) {
//	        // truncated or tampered .odz stream
//	}
package odzip
