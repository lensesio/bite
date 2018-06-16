package bite

import (
	"io/ioutil"
	"os"
)

// ReadInPipe reads from the input pipe.
//
// First argument returns true if in pipe has any data to read from,
// if false then the caller can continue by requiring a flag.
// Second argument returns the data of the io pipe,
// and third one is the error cames from .Stat() or from the ReadAll() of the in pipe.
func ReadInPipe() (bool, []byte, error) {
	// check if has data, otherwise it stucks.
	in := os.Stdin
	f, err := in.Stat()
	if err != nil {
		return false, nil, err
	}

	// check if has data is required, otherwise it stucks.
	if !(f.Mode()&os.ModeNamedPipe == 0) {
		b, err := ioutil.ReadAll(in)
		if err != nil {
			return true, nil, err
		}
		return true, b, nil
	}

	return false, nil, nil
}
