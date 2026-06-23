# go-odzip

[![Go Reference](https://pkg.go.dev/badge/github.com/odpay/go-odzip.svg)](https://pkg.go.dev/github.com/odpay/go-odzip)

Go bindings for [**libodzip**](https://github.com/odpay/odzip) — the LZ77 +
Huffman compressor behind the `odz` CLI and the [`.odz`](https://odzip.odpay.net)
format.

The C sources are vendored and built with cgo, so there is **no system library to
install** — only a C compiler (`cc`/`clang`/`gcc`), which Go uses automatically.
Streams produced by this package are byte-for-byte identical to those from the
`odz` CLI, and either tool can read the other's output.

## Install

```sh
go get github.com/odpay/go-odzip
```

Requires Go 1.21+, cgo enabled (the default), and a C toolchain on `PATH`.

## Usage

```go
import odzip "github.com/odpay/go-odzip"
```

**Files** — the most efficient path; libodzip streams in bounded memory:

```go
err := odzip.CompressFile("data.txt.odz", "data.txt", nil)   // dst, src
err  = odzip.DecompressFile("data.txt", "data.txt.odz", nil)
```

**Bytes:**

```go
packed, err := odzip.CompressBytes([]byte("hello hello hello"), nil)
plain,  err := odzip.DecompressBytes(packed, nil)
```

**Streams** (`io.Reader` / `io.Writer`):

```go
err := odzip.Compress(dst, src, nil)   // dst io.Writer, src io.Reader
err  = odzip.Decompress(dst, src, nil)
```

> Argument order follows `io.Copy`: **destination first, source second.**
> File and `*File` destinations are created or truncated.

### Options

A `nil *Options` selects the defaults (all cores, no progress callback).

```go
opts := &odzip.Options{
    Threads: 4, // <= 0 means runtime.NumCPU(); 1 forces single-threaded
    Progress: func(processed, total uint64) {
        fmt.Printf("\r%d / %d bytes", processed, total)
    },
}
err := odzip.CompressFile("big.odz", "big.bin", opts)
```

The thread count affects speed only — the compressed bytes are identical
regardless of it.

### Errors

Failures are reported as values of type `odzip.Error`. Match the sentinels with
`errors.Is`:

```go
if errors.Is(err, odzip.ErrCorrupt) {
    // truncated or tampered .odz stream
}
```

| Sentinel          | Meaning                                       |
| ----------------- | --------------------------------------------- |
| `odzip.ErrIO`     | I/O failure reading or writing a stream       |
| `odzip.ErrOOM`    | memory allocation failed                      |
| `odzip.ErrFormat` | bad magic or unsupported format version       |
| `odzip.ErrCorrupt`| data integrity error during decode            |

## CLI

A small command-line front-end is included:

```sh
go install github.com/odpay/go-odzip/cmd/odzip@latest

odzip c data.txt data.txt.odz   # compress
odzip d data.txt.odz data.txt   # decompress
odzip data.txt                  # auto-detect (-> data.txt.odz)
```

## Notes

* The `Progress` callback cannot cancel an operation — libodzip ignores the
  callback's return value. It is purely for reporting.
* The `[]byte` and stream helpers route through temporary files (libodzip
  requires seekable streams). For large or on-disk data, prefer the `*File`
  functions to avoid the extra copies.

## Layout & vendored C

```
odzip.go, doc.go        pure-Go public API (package odzip)
cmd/odzip/              installable CLI front-end
internal/libodzip/      all cgo + vendored libodzip C sources (quarantined here)
  ├─ shim.c/.h          hand-written cgo bridge
  ├─ libodzip.go        the cgo binding
  ├─ *.c / *.h          vendored verbatim from upstream odzip
  └─ UPSTREAM_REF       the exact odzip commit those sources came from
scripts/sync-upstream.sh   re-vendor from a pinned odzip ref
```

The C sources are **vendored, not a submodule** — Go's module proxy ships only a
module's own tracked files, so a submodule would arrive empty for `go get` and
the cgo build would fail. Instead the exact upstream commit is pinned in
[`internal/libodzip/UPSTREAM_REF`](internal/libodzip/UPSTREAM_REF), and
`scripts/sync-upstream.sh` reproduces or bumps it. The bindings track the odzip
**`v1.0.3`** release (the same artifact in Homebrew / the AUR). See
[`VENDOR.md`](VENDOR.md).

## License

MIT, matching upstream `odzip`. See [`LICENSE`](LICENSE).
