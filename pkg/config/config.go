package config

import (
	"encoding/json"
	"fmt"

	linotypes "github.com/lino-network/lino/types"
)

// RestAPI rest api endpoint and json path for response
type RestAPI struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	JSONPath string `json:"json_path"`
}

// Config is the config of price feeder.
type Config struct {
	ChainID          string    `json:"chain_id"`
	NodeURL          string    `json:"node_url"`
	MaxTxFeeLino     string    `json:"max_tx_fee_lino"`
	MaxRetry         int       `json:"max_retry"`
	RetryIntervalSec int64     `json:"retry_interval_sec"`
	FeedEverySec     int64     `json:"feed_every_sec"`
	RestAPIList      []RestAPI `json:"rest_api_list"`
}

func NewConfigFromBytes(bz []byte) (Config, error) {
	c := &Config{}
	err := json.Unmarshal(bz, c)
	return *c, err
}

// IsValid validates the config
func (c Config) IsValid() error {
	_, err := linotypes.LinoToCoin(c.MaxTxFeeLino)
	if err != nil {
		return err
	}
	if c.FeedEverySec < 60*10 || c.FeedEverySec >= 60*60 {
		return fmt.Errorf("Invalid feed every sec: %d,  valid: [10 minutes, 1 hour)  \n",
			c.FeedEverySec)
	}
	return nil
}

// TxFee returns the max transaction fee in config.
func (c Config) TxFee() linotypes.Coin {
	return linotypes.MustLinoToCoin(c.MaxTxFeeLino)
}

func (c Config) Bytes() []byte {
	bz, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return bz
}
