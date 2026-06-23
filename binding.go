package odzip

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

// Error is a libodzip status code returned when a compress or decompress
// operation fails. The package's exported sentinels (ErrIO, ErrOOM, ErrFormat,
// ErrCorrupt) are values of this type, so callers can match them with
// errors.Is:
//
//	if errors.Is(err, odzip.ErrCorrupt) { ... }
type Error int

// libodzip error codes. These mirror the ODZ_ERR_* constants in libodzip.h.
var (
	ErrIO      = Error(C.ODZ_ERR_IO)      // I/O failure reading or writing a stream
	ErrOOM     = Error(C.ODZ_ERR_OOM)     // memory allocation failed
	ErrFormat  = Error(C.ODZ_ERR_FORMAT)  // bad magic or unsupported format version
	ErrCorrupt = Error(C.ODZ_ERR_CORRUPT) // data integrity error during decode
)

// Error returns libodzip's canonical message for the code.
func (e Error) Error() string {
	return "odzip: " + C.GoString(C.odz_strerror(C.int(e)))
}

func errFromCode(code int) error {
	if code == C.ODZ_OK {
		return nil
	}
	return Error(code)
}

// progressRegistry bridges Go progress callbacks across the cgo boundary.
//
// cgo forbids passing a Go pointer (and hence a Go func) to C and holding it
// there. Instead we register the callback under an integer handle, hand the
// handle to C as opaque userdata, and the exported trampoline looks it back up.
type progressRegistry struct {
	mu   sync.Mutex
	next uintptr
	fns  map[uintptr]Progress
}

var registry = progressRegistry{fns: make(map[uintptr]Progress)}

// register stores fn and returns a nonzero handle, or 0 if fn is nil (which
// tells the C side to skip progress reporting entirely).
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

// compressPath / decompressPath are the thin cgo entry points. They take real
// filesystem paths because libodzip works on seekable FILE* streams.
func compressPath(dstPath, srcPath string, threads int, progress Progress) error {
	return runPath(dstPath, srcPath, threads, progress, true)
}

func decompressPath(dstPath, srcPath string, threads int, progress Progress) error {
	return runPath(dstPath, srcPath, threads, progress, false)
}

func runPath(dstPath, srcPath string, threads int, progress Progress, compress bool) error {
	cSrc := C.CString(srcPath)
	defer C.free(unsafe.Pointer(cSrc))
	cDst := C.CString(dstPath)
	defer C.free(unsafe.Pointer(cDst))

	h := registry.register(progress)
	defer registry.unregister(h)

	var rc C.int
	if compress {
		rc = C.odzgo_compress_path(cSrc, cDst, C.int(threads), C.intptr_t(h))
	} else {
		rc = C.odzgo_decompress_path(cSrc, cDst, C.int(threads), C.intptr_t(h))
	}
	return errFromCode(int(rc))
}
