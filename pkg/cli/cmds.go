package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/lino-network/lino/client"
	linotypes "github.com/lino-network/lino/types"
	"github.com/spf13/cobra"

	"github.com/lino-network/lino-price-feeder/pkg/config"
	"github.com/lino-network/lino-price-feeder/pkg/price"
)

const (
	FlagConfig = "config"
	FlagKey    = "priv-key"
)

func GetPriceCmd() *cobra.Command {
	configPath := ""
	cmd := &cobra.Command{
		Use:   "price",
		Short: "price --config <config_file> will query and print the current price",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configBytes, err := ioutil.ReadFile(configPath)
			if err != nil {
				return err
			}
			config, err := config.NewConfigFromBytes(configBytes)
			if err != nil {
				panic(err)
			}
			if err := config.IsValid(); err != nil {
				return err
			}

			pricer := price.NewMedianPricerFromConfig(config)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			priceList, err := pricer.PriceList(ctx)
			if err != nil {
				return err
			}
			pricer.PrintPriceList(priceList)
			return nil
		},
	}
	addConfigFlag(cmd, &configPath)
	return cmd
}

func GetFeedCmd() *cobra.Command {
	configPath := ""
	keyPath := ""

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "feed <validator> --config <config_file> --priv-key @<privkey_file>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(keyPath) == 0 {
				return fmt.Errorf("missing key path")
			}
			if keyPath[0] != '@' {
				return fmt.Errorf("Only encrypted private key file is allowed, i.e. @filepath." +
					"if you have typed the private key in plaintext, make sure remove the history.")
			}
			configBytes, err := ioutil.ReadFile(configPath)
			if err != nil {
				return err
			}
			cfg, err := config.NewConfigFromBytes(configBytes)
			if err != nil {
				panic(err)
			}
			if err := cfg.IsValid(); err != nil {
				return err
			}

			validator := linotypes.AccountKey(args[0])
			key, err := client.ParsePrivKey(keyPath)
			if err != nil {
				return err
			}

			feeder := NewFeeder(cfg, validator, key)
			return feeder.FeedLoop()
		},
	}

	addKeyFlag(cmd, &keyPath)
	addConfigFlag(cmd, &configPath)
	return cmd
}

func addConfigFlag(cmd *cobra.Command, configPath *string) {
	// config
	cmd.Flags().StringVar(configPath, FlagConfig, "c", "config file path of price feeder")
	_ = cmd.MarkFlagRequired(FlagConfig)
}

func addKeyFlag(cmd *cobra.Command, keyPath *string) {
	// secrets
	cmd.Flags().StringVar(keyPath, FlagKey, "", "encrypted private key file path")
	_ = cmd.MarkFlagRequired(FlagKey)
}
