package cli

import (
	"io"
	"os"
)

func stderrW() io.Writer { return os.Stderr }
