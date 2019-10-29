package price

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	linotypes "github.com/lino-network/lino/types"
	"github.com/tidwall/gjson"

	"github.com/lino-network/lino-price-feeder/pkg/config"
)

// Pricer returns the price of a coin.
type Pricer interface {
	Name() string
	Price(context.Context) (linotypes.MiniDollar, error)
}

type MedianPricer struct {
	Sources []Pricer
}

func NewMedianPricerFromConfig(cfg config.Config) (rst MedianPricer) {
	for _, api := range cfg.RestAPIList {
		rst.Sources = append(rst.Sources, RestAPIPricer{c: api})
	}
	return
}

func (m MedianPricer) Name() string {
	name := "median of ["
	sep := ""
	for _, n := range m.Sources {
		name += sep
		name += n.Name()
		sep = ","
	}
	name += "]"
	return name
}

func (m MedianPricer) Price(ctx context.Context) (linotypes.MiniDollar, error) {
	prices := make([]linotypes.MiniDollar, len(m.Sources))
	for i, source := range m.Sources {
		var err error
		prices[i], err = source.Price(ctx)
		if err != nil {
			return linotypes.MiniDollar{}, err
		}
	}
	sort.Slice(prices, func(i int, j int) bool {
		return prices[i].LT(prices[j])
	})
	// do not take the mean of two, use higher one.
	return prices[len(prices)/2], nil
}

func (m MedianPricer) PriceList(ctx context.Context) (map[string]linotypes.MiniDollar, error) {
	rst := make(map[string]linotypes.MiniDollar)
	for _, source := range m.Sources {
		var err error
		rst[source.Name()], err = source.Price(ctx)
		if err != nil {
			return rst, err
		}
	}
	return rst, nil
}

func (m MedianPricer) PrintPriceList(rst map[string]linotypes.MiniDollar) {
	fmt.Printf("|%-10s|%-25s|%-18s|\n",
		"name", "price (Coin/MiniDollar)", "price (LINO/USD)")
	for name, value := range rst {
		vI64 := value.Int64()
		uprice := float64(vI64) / linotypes.Decimals
		fmt.Printf("|%-10s|%-25s|%-18s|\n",
			name, value, strconv.FormatFloat(uprice, 'f', 8, 64))
	}
}

func (m MedianPricer) PriceFromPriceList(lst map[string]linotypes.MiniDollar) linotypes.MiniDollar {
	if len(lst) == 0 {
		return linotypes.NewMiniDollar(0)
	}
	prices := make([]linotypes.MiniDollar, 0)
	for _, p := range lst {
		prices = append(prices, p)
	}
	sort.Slice(prices, func(i int, j int) bool {
		return prices[i].LT(prices[j])
	})
	// do not take the mean of two, use higher one.
	return prices[len(prices)/2]
}

type RestAPIPricer struct {
	c config.RestAPI
}

func (r RestAPIPricer) Name() string {
	return r.c.Name
}

func (r RestAPIPricer) Price(ctx context.Context) (rst linotypes.MiniDollar, err error) {
	req, err := http.NewRequest("GET", r.c.Endpoint, nil)
	if err != nil {
		return rst, err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rst, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rst, err
	}
	json := string(body)
	if !gjson.Valid(json) {
		return rst, fmt.Errorf("invalid json response: %s", r.c.Name)
	}
	value := gjson.Get(string(body), r.c.JSONPath).Value()
	switch v := value.(type) {
	case string:
		rst, err = usdStrToMiniDollar(v)
		if err != nil {
			return rst, err
		}
	case float64:
		rst = usdFloat64ToMiniDollar(v)
	default:
		return rst, fmt.Errorf("Unknown value of %+v, value: %+v, body: %s", r.c, value, json)
	}

	if !rst.IsPositive() {
		return rst, fmt.Errorf("invalid price of %s, non-positive, %v", r.c.Name, rst)
	}
	return rst, nil
}

func usdStrToMiniDollar(str string) (rst linotypes.MiniDollar, err error) {
	dec, err := sdk.NewDecFromStr(str)
	if err != nil {
		return rst, err
	}
	// mini dollar is 10^(-10) dollar.
	// one coin is 10^(-5) Lino.
	// e.g.
	// lino / usd = 0.012
	// coin / minidollar = 1200
	return linotypes.NewMiniDollarFromInt(
		dec.MulInt64(linotypes.Decimals).TruncateInt()), nil
}

func usdFloat64ToMiniDollar(val float64) linotypes.MiniDollar {
	rst, _ := usdStrToMiniDollar(strconv.FormatFloat(val, 'f', 10, 64))
	return rst
}
