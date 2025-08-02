package command

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"terrasync/app/scan"
)

func NewScanCommand(AppVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan <storage>",
		Short: "Scan storage system to get the statistics.",
		Long:  "Read all the files in a file tree and create a report based on the options.",
		Example: `  
    Scan without options:
      terrasync scan <scanPath>

    Show statistics in scan output:
	  terrasync scan --stats <scanPath>

	Scan up to depth of 4. Depth -1 lists all subdirectories:
	  terrasync scan --depth 4 <scanPath>
	
	Create HTML or CSV report (not yet implemented):
	 terrasync scan --html <scanPath>
	
	Print a report to the console for files matching criteria:
	  terrasync scan --stats --match 'owner=="root" and size>100M' <scanPath>

	List regular files with "ntap" in the name and modified in the last half hour:
	  terrasync -l --match 'modified<0.5 and "ntap" in name and type==file' <scanPath>

	Exclude files modified less than half an hour ago:
	 terrasync scan -exclude "type==file and modified<0.5" <scanPath>
	
	Run scan with loglevel(default: INFO) to generate debug logs:
	  terrasync scan --loglevel DEBUG  <scanPath>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build full command line string
			cmdLine := buildCommandLine(cmd, args)

			// Get executable path and directory
			goexe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}
			goexeDir := filepath.Dir(goexe)

			// Read database type from config file
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
			viper.AddConfigPath(goexeDir)
			if err = viper.ReadInConfig(); err != nil {
				return fmt.Errorf("error reading config file: %w", err)
			}

			concurrency := viper.GetInt("scan.concurrency")
			dbType := viper.GetString("database.type")
			dbBatchSize := viper.GetInt("database.batch_size")

			// Read Kafka configuration
			kafkaEnabled := viper.GetBool("kafka.enabled")
			kafkaTopic := viper.GetString("kafka.topic")
			kafkaHost := viper.GetString("kafka.host")
			kafkaPort := viper.GetInt("kafka.port")
			kafkaConcurrency := viper.GetInt("kafka.concurrency")

			scanID, _ := cmd.Flags().GetString("id")
			depth, _ := cmd.Flags().GetInt("depth")
			matchExpr, _ := cmd.Flags().GetString("match")
			excludeExpr, _ := cmd.Flags().GetString("exclude")
			csvReport, _ := cmd.Flags().GetBool("csv")
			htmlReport, _ := cmd.Flags().GetBool("html")
			quiet, _ := cmd.Flags().GetBool("quiet")

			var jobID string
			if scanID == "" {
				// Generate job ID in the format: Job_YYYY-MM-DD_HH.MM.SS.ffffff_scan
				jobID = fmt.Sprintf("Job_%s_scan", time.Now().Format("2006-01-02_15.04.05.000000"))
			} else {
				jobID = fmt.Sprintf("Job_%s_scan", scanID)
			}
			// Set up job directory
			jobsDir, incrementalScan, err := isIncrementalScan(jobID, goexeDir)
			if err != nil {
				return err
			}

			scanPath := args[0]

			// 创建扫描配置结构体
			scanConfig := scan.ScanConfig{
				IncrementalScan: incrementalScan,
				JobDir:          jobsDir,
				DBBatchSize:     dbBatchSize,
				DbType:          dbType,
				Path:            scanPath,
				Concurrency:     concurrency,
				Depth:           depth,
				Match:           scan.ParseConditions(matchExpr),
				Exclude:         scan.ParseConditions(excludeExpr),
			}

			reportConfig := scan.ReportConfig{
				AppVersion: AppVersion,
				CmdLine:    cmdLine,
				CsvReport:  csvReport,
				HtmlReport: htmlReport,
				JobID:      jobID,
				LogPath:    filepath.Join(goexeDir, "terrasync.log"),
				StartTime:  time.Now(),
				KafkaConfig: scan.KafkaConfig{
					Enabled:     kafkaEnabled,
					Topic:       kafkaTopic,
					Host:        kafkaHost,
					Port:        kafkaPort,
					Concurrency: kafkaConcurrency,
				},
				Quiet: quiet,
			}

			if err := scan.Start(scanConfig, reportConfig); err != nil {
				return fmt.Errorf("failed to scan: %w", err)
			}

			return nil
		},
	}

	// Add command line flags
	cmd.Flags().StringP("id", "", "", "Job id for scan, special for incremental scan")
	cmd.Flags().IntP("depth", "d", 0, "Set maximum scan depth")
	cmd.Flags().StringP("match", "m", "", "Filter files using the given expression")
	cmd.Flags().StringP("exclude", "e", "", "Exclude files using the given expression")
	cmd.Flags().BoolP("csv", "", false, "Create CSV report")
	cmd.Flags().BoolP("html", "", false, "Create HTML report")
	cmd.Flags().BoolP("quiet", "q", false, "no output in the console, but in the log.")

	return cmd
}
