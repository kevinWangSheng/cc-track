// Generator 模板：新增 cobra 命令时复制此模板填空

package cmd

import (
	"fmt"

	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var {{.Name}}Cmd = &cobra.Command{
	Use:   "{{.Use}}",
	Short: "{{.Short}}",
	RunE:  run{{.Name}},
}

func init() {
	rootCmd.AddCommand({{.Name}}Cmd)
	// {{.Name}}Cmd.Flags().Bool("flag-name", false, "description")
}

func run{{.Name}}(cmd *cobra.Command, args []string) error {
	db, err := store.Open(config.DBPath())
	if err != nil {
		return fmt.Errorf("{{.Use}}: %w", err)
	}
	defer db.Close()

	jsonMode, _ := cmd.Flags().GetBool("json")

	// TODO: 业务逻辑

	if jsonMode {
		return output.JSON(cmd.OutOrStdout(), result)
	}
	return output.Table(cmd.OutOrStdout(), result)
}
