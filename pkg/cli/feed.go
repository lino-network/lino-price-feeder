package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	linoapi "github.com/lino-network/lino-go/api"
	linotypes "github.com/lino-network/lino/types"
	crypto "github.com/tendermint/tendermint/crypto"

	"github.com/lino-network/lino-price-feeder/pkg/config"
	"github.com/lino-network/lino-price-feeder/pkg/price"
)

var errRevoked = fmt.Errorf("the validator is revoked")
var errKeyMismatch = fmt.Errorf("provided private key does not match user's account info")

type Feeder struct {
	api      *linoapi.API
	username linotypes.AccountKey
	key      crypto.PrivKey
	cfg      config.Config
	pricer   price.MedianPricer
}

func NewFeeder(cfg config.Config, validator linotypes.AccountKey, key crypto.PrivKey) Feeder {
	fees := linotypes.MustLinoToCoin(cfg.MaxTxFeeLino)
	feesI64, err := fees.ToInt64()
	if err != nil {
		panic(err)
	}
	api := linoapi.NewLinoAPIFromArgs(&linoapi.Options{
		ChainID:      cfg.ChainID,
		NodeURL:      cfg.NodeURL,
		MaxFeeInCoin: feesI64,
	})

	pricer := price.NewMedianPricerFromConfig(cfg)

	return Feeder{
		api:      api,
		username: validator,
		key:      key,
		cfg:      cfg,
		pricer:   pricer,
	}
}

func (f Feeder) isValid() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	validator, err := f.api.GetValidator(ctx, string(f.username))
	if err != nil {
		return err
	}
	if validator.HasRevoked {
		return errRevoked
	}

	info, err := f.api.GetAccountInfo(ctx, string(f.username))
	if err != nil {
		return err
	}
	if !(info.SigningKey.Equals(f.key.PubKey()) || info.TransactionKey.Equals(f.key.PubKey())) {
		return errKeyMismatch
	}

	// try get price
	_, err = f.pricer.Price(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (f Feeder) feed() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// query last feed time.
	last, _ := f.api.GetLastFeed(ctx, string(f.username))

	// skip if has fed recently.
	now := time.Now().Unix()
	if last != nil && now-last.UpdateAt < f.cfg.FeedEverySec {
		fmt.Printf("now(%d) - %s's last feed time(%d) = %d <= %d seconds, skipped.\n",
			now, f.username, last.UpdateAt, now-last.UpdateAt, f.cfg.FeedEverySec)
		return nil
	}

	// skip if not in oncall or standby list.
	valList, err := f.api.GetAllValidators(ctx)
	if err != nil {
		return err
	}
	if linotypes.FindAccountInList(f.username, valList.Oncall) == -1 &&
		linotypes.FindAccountInList(f.username, valList.Standby) == -1 {
		fmt.Printf("%s is not in oncall or standby list, skipping. \n", f.username)
		return nil
	}

	// feed
	priceList, err := f.pricer.PriceList(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Price List:\n")
	f.pricer.PrintPriceList(priceList)
	price := f.pricer.PriceFromPriceList(priceList)
	resp, err := f.api.FeedPrice(ctx, string(f.username), price, hex.EncodeToString(f.key.Bytes()))
	if err != nil {
		return err
	}
	fmt.Printf(
		"feed 1 coin = %d minidollar, i.e. 1 LINO = %f$; Height %d, TX: %s\n",
		price.Int64(), float64(price.Int64())/linotypes.Decimals, resp.Height, resp.CommitHash)
	return nil
}

func (f Feeder) FeedLoop() error {
	if err := f.isValid(); err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	err := f.feed()
	if err != nil {
		return err
	}

	// TODO(yumin): better error handling.
	for {
		select {
		case <-sigint:
			return nil
		case <-ticker.C:
			nFailed := 0
			for {
				err := f.feed()
				if err != nil {
					fmt.Printf("Feed Error: %s\n", err)
					nFailed++
					if nFailed > f.cfg.MaxRetry {
						return err
					}
					time.Sleep(time.Duration(f.cfg.RetryIntervalSec) * time.Second)
				} else {
					break
				}
			}
		}
	}
}
