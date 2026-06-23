# Vendored C sources

This module embeds the C sources of [libodzip](https://github.com/odpay/odzip)
under `internal/libodzip/` so it builds with cgo and is `go get`-able with no
separate native build step. The vendored files are copied **verbatim** from
upstream — do not edit them here.

## Why vendor instead of a git submodule?

Go modules are distributed by the module proxy as a zip of the module's own
tracked files. **Git submodule contents are not included**, so a `go get` of this
module would pull an *empty* submodule directory and the cgo build would fail for
lack of the `.c`/`.h` files. Vendoring is therefore required for any
`go get`-able cgo library. To keep the copy honest and easy to refresh, the
exact upstream revision is pinned and a sync script reproduces it.

## Pinned upstream revision

`internal/libodzip/UPSTREAM_REF` records the exact odzip commit the sources were
copied from. The current pin is:

```
4bddebc4d1f47f767188c367f4485905adef9852   # v1.0.3 + 3 (incl. the decompress OOM hardening fix)
```

The public API (`libodzip.h`: `odz_compress`, `odz_decompress`, `odz_strerror`,
`odz_options_t`, and the `ODZ_*` codes) is identical to the `v1.0.3` release; the
pin is 3 commits ahead only to include the decompress `comp_size > ODZ_BLOCK_SIZE`
hardening. The vendored sources track `.odz` **format version 2**
(`ODZ_FORMAT_VERSION`).

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

Hand-written, not vendored: `shim.c`, `shim.h` (the cgo bridge), and
`libodzip.go` (the cgo binding).

## Refreshing from upstream

```sh
scripts/sync-upstream.sh            # re-copy at the pinned UPSTREAM_REF
scripts/sync-upstream.sh v1.0.4     # bump to a new tag/branch/sha and re-pin
```

Then review `git diff`, run `go test ./...` (which round-trips data and checks
the `odzip.Error` mapping), and — if upstream changed `libodzip.h` — update
`shim.c`/`shim.h`/`libodzip.go` to match before committing.
