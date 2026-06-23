package odzip_test

import (
	"fmt"

	odzip "github.com/odpay/go-odzip"
)

func Example() {
	original := []byte("compress me, compress me, compress me, please")

	packed, err := odzip.CompressBytes(original, nil)
	if err != nil {
		panic(err)
	}

	restored, err := odzip.DecompressBytes(packed, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", restored)
	// Output: compress me, compress me, compress me, please
}

func ExampleOptions() {
	// Run single-threaded and report progress as work proceeds.
	opts := &odzip.Options{
		Threads: 1,
		Progress: func(processed, total uint64) {
			_ = processed
			_ = total
		},
	}
	packed, _ := odzip.CompressBytes([]byte("hello hello hello hello"), opts)
	restored, _ := odzip.DecompressBytes(packed, opts)

	fmt.Printf("%s", restored)
	// Output: hello hello hello hello
}
