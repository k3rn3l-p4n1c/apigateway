package version

import (
	"fmt"
	"io"
	"os"
)

func FprintVersion(w io.Writer) {
	fmt.Fprintln(w, Package, fmt.Sprintf("%d.%d.%d", Major, Minor, Patch))
}

// PrintVersion outputs the version information, from Fprint, to stdout.
func PrintVersion() {
	FprintVersion(os.Stdout)
}
