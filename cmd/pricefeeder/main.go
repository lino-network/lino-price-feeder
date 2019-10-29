package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lino-network/lino-price-feeder/pkg/cli"
)

var rootCmd = &cobra.Command{
	Use:   "lino-price-feeder",
	Short: "Query and feed current price of LINO",
}

func main() {
	cobra.EnableCommandSorting = false
	rootCmd.AddCommand(
		cli.GetFeedCmd(),
		cli.GetPriceCmd())
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %+v\n", err)
		os.Exit(1)
	}
}
