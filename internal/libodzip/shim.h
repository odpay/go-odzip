/*
 * shim.h — C bridge between Go (cgo) and libodzip.
 *
 * libodzip's public API operates on stdio FILE* streams. cgo cannot hand a Go
 * io.Reader/Writer to C as a FILE*, and the portable in-memory FILE helpers
 * (fmemopen/open_memstream) are not available on macOS or Windows. So the Go
 * layer always materialises its work as real files and these helpers open them
 * with fopen(), which is seekable and portable everywhere libodzip builds.
 *
 * `handle` is an opaque token the Go side uses to look up a progress callback.
 * Pass 0 to disable progress reporting.
 */
#ifndef ODZGO_SHIM_H
#define ODZGO_SHIM_H

#include <stdint.h>

int odzgo_compress_path(const char *in_path, const char *out_path,
                        int threads, intptr_t handle);
int odzgo_decompress_path(const char *in_path, const char *out_path,
                          int threads, intptr_t handle);

#endif /* ODZGO_SHIM_H */
