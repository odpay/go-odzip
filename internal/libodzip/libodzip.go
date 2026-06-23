// Package libodzip is the cgo binding for the vendored libodzip C sources.
// It is internal: the public, idiomatic API lives in the root odzip package,
// which is pure Go. All cgo and all C code are quarantined here.
//
// The C sources in this directory are vendored verbatim from upstream odzip;
// see VENDOR.md and scripts/sync-upstream.sh. Only shim.c, shim.h, and this
// file are hand-written.
package libodzip

/*
#cgo CFLAGS: -O2 -DNDEBUG -I${SRCDIR}
#cgo !windows CFLAGS: -pthread
#cgo !windows LDFLAGS: -pthread

#include <stdlib.h>
#include "libodzip.h"
#include "shim.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

// libodzip status codes, mirrored from libodzip.h.
var (
	CodeOK      = int(C.ODZ_OK)
	CodeIO      = int(C.ODZ_ERR_IO)
	CodeOOM     = int(C.ODZ_ERR_OOM)
	CodeFormat  = int(C.ODZ_ERR_FORMAT)
	CodeCorrupt = int(C.ODZ_ERR_CORRUPT)
)

// Strerror returns libodzip's canonical message for a status code.
func Strerror(code int) string {
	return C.GoString(C.odz_strerror(C.int(code)))
}

// Progress mirrors odzip.Progress; see that type for semantics.
type Progress func(processed, total uint64)

// progressRegistry bridges Go progress callbacks across the cgo boundary. cgo
// forbids parking a Go func pointer in C, so we register the callback under an
// integer handle, pass the handle to C as opaque userdata, and the exported
// trampoline looks it back up.
type progressRegistry struct {
	mu   sync.Mutex
	next uintptr
	fns  map[uintptr]Progress
}

var registry = progressRegistry{fns: make(map[uintptr]Progress)}

func (r *progressRegistry) register(fn Progress) uintptr {
	if fn == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	if r.next == 0 { // never hand out 0; it means "no callback"
		r.next++
	}
	h := r.next
	r.fns[h] = fn
	return h
}

func (r *progressRegistry) unregister(h uintptr) {
	if h == 0 {
		return
	}
	r.mu.Lock()
	delete(r.fns, h)
	r.mu.Unlock()
}

func (r *progressRegistry) lookup(h uintptr) Progress {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fns[h]
}

//export goOdzProgress
func goOdzProgress(handle C.intptr_t, processed, total C.uint64_t) C.int {
	if fn := registry.lookup(uintptr(handle)); fn != nil {
		fn(uint64(processed), uint64(total))
	}
	// libodzip ignores this value; 0 means "continue".
	return 0
}

// Compress compresses srcPath into dstPath. It returns a libodzip status code
// (CodeOK on success). dstPath is created or truncated.
func Compress(dstPath, srcPath string, threads int, progress Progress) int {
	return run(dstPath, srcPath, threads, progress, true)
}

// Decompress decompresses srcPath into dstPath. It returns a libodzip status
// code (CodeOK on success). dstPath is created or truncated.
func Decompress(dstPath, srcPath string, threads int, progress Progress) int {
	return run(dstPath, srcPath, threads, progress, false)
}

func run(dstPath, srcPath string, threads int, progress Progress, compress bool) int {
	cSrc := C.CString(srcPath)
	defer C.free(unsafe.Pointer(cSrc))
	cDst := C.CString(dstPath)
	defer C.free(unsafe.Pointer(cDst))

	h := registry.register(progress)
	defer registry.unregister(h)

	if compress {
		return int(C.odzgo_compress_path(cSrc, cDst, C.int(threads), C.intptr_t(h)))
	}
	return int(C.odzgo_decompress_path(cSrc, cDst, C.int(threads), C.intptr_t(h)))
}
