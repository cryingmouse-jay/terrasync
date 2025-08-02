package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"terrasync/log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// isIncrementalScan checks if a job directory exists and returns true if it does
func isIncrementalScan(jobID, exeDir string) (string, bool, error) {
	jobsDir := filepath.Join(exeDir, "jobs", jobID)

	// Check if directory exists
	_, err := os.Stat(jobsDir)
	if err == nil {
		// Directory exists, mark as incremental scan
		return jobsDir, true, nil
	} else if !os.IsNotExist(err) {
		// Other error occurred
		log.Errorf("Failed to check directory %s: %v", jobsDir, err)
		return "", false, err
	}

	// Directory does not exist, create it and mark as full scan
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		log.Errorf("Failed to create directory %s: %v", jobsDir, err)
		return "", false, err
	}

	return jobsDir, false, nil
}

// buildCommandLine constructs a complete command line string
func buildCommandLine(cmd *cobra.Command, args []string) string {
	var sb strings.Builder
	sb.WriteString(cmd.CommandPath())

	// Add all set flags
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			if f.Shorthand != "" {
				sb.WriteString(" -" + f.Shorthand)
			} else {
				sb.WriteString(" --" + f.Name)
			}
			// Add value for non-boolean flags
			if f.Value.Type() != "bool" {
				sb.WriteString(" " + fmt.Sprintf("%q", f.Value.String()))
			}
		}
	})

	// Add positional arguments
	for _, arg := range args {
		sb.WriteString(" " + fmt.Sprintf("%q", arg))
	}

	return sb.String()
}
