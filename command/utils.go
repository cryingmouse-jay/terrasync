package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"terrasync/log"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// setupJobDirectory creates a clean job directory
func setupJobDirectory(jobID, exeDir string) (string, error) {
	if jobID == "" {
		// Generate timestamp-based jobID (year-month-day_hour.minute.second)
		jobID = time.Now().Format("2006-01-02_15.04.05")
	}

	jobsDir := filepath.Join(exeDir, "jobs", jobID)

	// Clear directory if it exists
	if err := os.RemoveAll(jobsDir); err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to clear directory %s: %v", jobsDir, err)
		return "", err
	}

	// Create directory with appropriate permissions
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		log.Errorf("Failed to create directory %s: %v", jobsDir, err)
		return "", err
	}

	return jobsDir, nil
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
