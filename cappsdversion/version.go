package cappsdversion

import (
	"fmt"
	"io"
	"os"
)

// Version variables linked at compile time.
var (
	Version    = "unknown"
	GitCommit  = "unknown"
	BuildStamp = "unknown"
)

// FprintVersion writes version info to specified writer
func FprintVersion(out io.Writer) {
	fmt.Fprintf(out, "ems version:  %s\n", Version)
	fmt.Fprintf(out, "    commit:   %s\n", GitCommit)
	fmt.Fprintf(out, "    buildUTC: %s\n", BuildStamp)
}

// PrintVersion writes version info to stdout
func PrintVersion() {
	FprintVersion(os.Stdout)
}
