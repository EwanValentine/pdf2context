package deps

import (
	"fmt"
	"os/exec"
	"strings"
)

type tool struct {
	name        string
	macInstall  string
	linuxInstall string
}

var required = []tool{
	{
		name:         "pdftotext",
		macInstall:   "brew install poppler",
		linuxInstall: "apt-get install poppler-utils",
	},
	{
		name:         "ocrmypdf",
		macInstall:   "brew install ocrmypdf",
		linuxInstall: "apt-get install ocrmypdf",
	},
}

// Check verifies that all required external tools are available in PATH.
// If any are missing, it returns a descriptive error with install hints.
func Check() error {
	var missing []tool
	for _, t := range required {
		if _, err := exec.LookPath(t.name); err != nil {
			missing = append(missing, t)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("missing required tools:\n")
	for _, t := range missing {
		sb.WriteString(fmt.Sprintf("  - %s\n", t.name))
		sb.WriteString(fmt.Sprintf("    macOS:  %s\n", t.macInstall))
		sb.WriteString(fmt.Sprintf("    Linux:  %s\n", t.linuxInstall))
	}
	return fmt.Errorf("%s", sb.String())
}
