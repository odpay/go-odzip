/*
 * shim.c — implementation of the cgo bridge declared in shim.h.
 *
 * Each entry point opens the input/output files, wires up an optional progress
 * trampoline, and forwards to libodzip. The trampoline calls back into Go via
 * goOdzProgress (defined in binding.go and exported by cgo). Note: libodzip
 * ignores the progress return value, so cancellation is not supported — the
 * trampoline always returns 0 ("continue").
 */
#include <stdio.h>
#include <stdint.h>

#include "libodzip.h"
#include "shim.h"

/* Defined in Go, exported via cgo's //export directive. */
extern int goOdzProgress(intptr_t handle, uint64_t processed, uint64_t total);

static int odzgo_progress_tramp(uint64_t processed, uint64_t total, void *ud) {
    return goOdzProgress((intptr_t)ud, processed, total);
}

static int odzgo_run(const char *in_path, const char *out_path,
                     int threads, intptr_t handle, int compress) {
    FILE *in = fopen(in_path, "rb");
    if (!in) return ODZ_ERR_IO;

    FILE *out = fopen(out_path, "wb");
    if (!out) { fclose(in); return ODZ_ERR_IO; }

    odz_options_t opts;
    opts.progress = handle ? odzgo_progress_tramp : NULL;
    opts.userdata = (void *)handle;
    opts.threads  = threads;

    int rc = compress ? odz_compress(in, out, &opts)
                      : odz_decompress(in, out, &opts);

    fclose(in);
    /* A failed final flush/close means the output is not trustworthy. */
    if (fclose(out) != 0 && rc == ODZ_OK)
        rc = ODZ_ERR_IO;

    return rc;
}

int odzgo_compress_path(const char *in_path, const char *out_path,
                        int threads, intptr_t handle) {
    return odzgo_run(in_path, out_path, threads, handle, 1);
}

int odzgo_decompress_path(const char *in_path, const char *out_path,
                          int threads, intptr_t handle) {
    return odzgo_run(in_path, out_path, threads, handle, 0);
}
