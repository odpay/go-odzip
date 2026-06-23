# Vendored C sources

This module embeds the C sources of [libodzip](https://github.com/odpay/odzip)
so it builds with cgo and is `go get`-able without a separate native build step.
The vendored files are copied **verbatim** from upstream — do not edit them here.
All Go-facing bridging lives in `shim.c` / `shim.h`, which are part of this repo
and not vendored.

## Files vendored from odpay/odzip

Sources (compiled by cgo as separate translation units, exactly as upstream's
CMake `LIB_SOURCES` does):

```
odz_util.c  bitstream.c  huffman.c  lz_hashchain.c  compress.c  decompress.c
```

Headers:

```
libodzip.h  odz.h  odz_thread.h  bitstream.h  huffman.h  lz_matcher.h  lz_tables.h
```

`main.c` is intentionally **not** vendored — it defines `main()` and is the CLI,
not the library.

## Refreshing from upstream

From a checkout of `odpay/odzip` at the desired revision:

```sh
SRC=/path/to/odzip
DST=/path/to/go-odzip
for f in odz_util.c bitstream.c huffman.c lz_hashchain.c compress.c decompress.c \
         libodzip.h odz.h odz_thread.h bitstream.h huffman.h lz_matcher.h lz_tables.h; do
  cp "$SRC/$f" "$DST/$f"
done
```

Then run `go test ./...` (which round-trips data and checks `odzip.Error`
mapping). If upstream changes the public API in `libodzip.h` — currently
`odz_compress`, `odz_decompress`, `odz_strerror`, the `odz_options_t` struct, and
the `ODZ_*` error codes — update `shim.c`/`shim.h` and `binding.go` to match.

The current vendor tracks `.odz` **format version 2** (`ODZ_FORMAT_VERSION`).
