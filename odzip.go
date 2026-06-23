package odzip

import (
	"bytes"
	"io"
	"os"
	"runtime"

	"github.com/odpay/go-odzip/internal/libodzip"
)

// Progress is an optional callback invoked while an operation runs. processed
// is the number of input bytes handled so far and total is the full input size
// (which may be 0 for empty input). Keep it cheap: it is called synchronously
// from inside the C library. Cancellation is not supported by libodzip, so the
// callback cannot abort an operation.
type Progress func(processed, total uint64)

// Options configures a compress or decompress operation. A nil *Options is
// always valid and selects the defaults.
type Options struct {
	// Threads sets libodzip's worker thread count. A value <= 0 selects an
	// automatic default of runtime.NumCPU(); 1 forces single-threaded. The
	// thread count affects speed only — the bytes produced are identical
	// regardless of it. Values above 256 are clamped.
	Threads int

	// Progress, if non-nil, receives periodic progress updates. See Progress.
	Progress Progress
}

// Error is a libodzip status code returned when an operation fails. The
// package's exported sentinels (ErrIO, ErrOOM, ErrFormat, ErrCorrupt) are
// values of this type, so callers can match them with errors.Is:
//
//	if errors.Is(err, odzip.ErrCorrupt) { ... }
type Error int

// libodzip error codes. These mirror the ODZ_ERR_* constants in libodzip.h.
var (
	ErrIO      = Error(libodzip.CodeIO)      // I/O failure reading or writing a stream
	ErrOOM     = Error(libodzip.CodeOOM)     // memory allocation failed
	ErrFormat  = Error(libodzip.CodeFormat)  // bad magic or unsupported format version
	ErrCorrupt = Error(libodzip.CodeCorrupt) // data integrity error during decode
)

// Error returns libodzip's canonical message for the code.
func (e Error) Error() string {
	return "odzip: " + libodzip.Strerror(int(e))
}

func errFromCode(code int) error {
	if code == libodzip.CodeOK {
		return nil
	}
	return Error(code)
}

func resolveThreads(opts *Options) int {
	n := 0
	if opts != nil {
		n = opts.Threads
	}
	if n <= 0 {
		n = runtime.NumCPU()
	}
	if n > 256 {
		n = 256
	}
	return n
}

func progressOf(opts *Options) libodzip.Progress {
	if opts == nil || opts.Progress == nil {
		return nil
	}
	return libodzip.Progress(opts.Progress)
}

// CompressFile compresses srcPath into the .odz stream written to dstPath.
//
// Argument order follows io.Copy: destination first, source second. dstPath is
// created or truncated. This is the most efficient entry point — libodzip
// streams both files in bounded memory with no intermediate copies.
func CompressFile(dstPath, srcPath string, opts *Options) error {
	return errFromCode(libodzip.Compress(dstPath, srcPath, resolveThreads(opts), progressOf(opts)))
}

// DecompressFile decompresses the .odz stream at srcPath into dstPath.
//
// Argument order follows io.Copy: destination first, source second. dstPath is
// created or truncated.
func DecompressFile(dstPath, srcPath string, opts *Options) error {
	return errFromCode(libodzip.Decompress(dstPath, srcPath, resolveThreads(opts), progressOf(opts)))
}

// Compress reads all of src, compresses it, and writes the .odz stream to dst.
//
// libodzip operates on seekable files, so this helper spools src and the result
// through temporary files (see os.TempDir). For large or already-on-disk data,
// prefer CompressFile to avoid the extra copies.
func Compress(dst io.Writer, src io.Reader, opts *Options) error {
	return viaTempFiles(dst, src, opts, CompressFile)
}

// Decompress reads the .odz stream from src, decompresses it, and writes the
// result to dst. Like Compress, it works through temporary files; prefer
// DecompressFile when your data is already a file on disk.
func Decompress(dst io.Writer, src io.Reader, opts *Options) error {
	return viaTempFiles(dst, src, opts, DecompressFile)
}

// CompressBytes compresses src and returns the .odz stream.
func CompressBytes(src []byte, opts *Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := Compress(&buf, bytes.NewReader(src), opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressBytes decompresses an .odz stream and returns the original bytes.
func DecompressBytes(src []byte, opts *Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := Decompress(&buf, bytes.NewReader(src), opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// viaTempFiles materialises an io.Reader/io.Writer operation as a file→file one,
// since libodzip requires seekable streams.
func viaTempFiles(dst io.Writer, src io.Reader, opts *Options, op func(dstPath, srcPath string, opts *Options) error) (err error) {
	inFile, err := os.CreateTemp("", "odzip-in-*")
	if err != nil {
		return err
	}
	inPath := inFile.Name()
	defer os.Remove(inPath)

	if _, err = io.Copy(inFile, src); err != nil {
		inFile.Close()
		return err
	}
	if err = inFile.Close(); err != nil {
		return err
	}

	// Pre-create the output path so we control cleanup; op reopens it by name.
	outFile, err := os.CreateTemp("", "odzip-out-*")
	if err != nil {
		return err
	}
	outPath := outFile.Name()
	defer os.Remove(outPath)
	if err = outFile.Close(); err != nil {
		return err
	}

	if err = op(outPath, inPath, opts); err != nil {
		return err
	}

	result, err := os.Open(outPath)
	if err != nil {
		return err
	}
	defer result.Close()

	_, err = io.Copy(dst, result)
	return err
}
