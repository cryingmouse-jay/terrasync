package command

import (
	"fmt"
	"terrasync/object"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewMigrateCommand creates migration command
func NewMigrateCommand(AppVersion string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "migrate <source> <destination>",
		Short: "Migrate data from source to destination",
		Long:  "Migrate data from source to destination, supporting multiple storage types, such as CIFS, NFS, S3.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			dst := args[1]

			// 从Viper获取配置，命令行参数优先级更高
			overwrite := viper.GetBool("migrate.overwrite")
			threads := viper.GetInt("migrate.concurrency")

			fmt.Print(overwrite)
			fmt.Print(threads)

			srcStorage, err := object.CreateStorage(src)
			if err != nil {
				return fmt.Errorf("failed to create source storage: %w", err)
			}
			defer srcStorage.Close()

			dstStorage, err := object.CreateStorage(dst)
			if err != nil {
				return fmt.Errorf("failed to create destination storage: %w", err)
			}
			defer dstStorage.Close()

			// TODO: Implement actual data migration logic

			return nil
		},
	}

	// Add command line flags
	cmd.Flags().BoolP("overwrite", "", false, "Overwrite the existing files in destination storage")
	cmd.Flags().IntP("concurrency", "", 5, "Concurrency threads for migration")

	return cmd
}
