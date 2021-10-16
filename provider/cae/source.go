package cae

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider"
	"github.com/robotomize/gokuu/provider/httputil"
)

const hostname = "www.centralbank.ae"

var exchangeableSymbols = []label.Symbol{
	label.AED, label.USD, label.ARS, label.AUD, label.BND, label.BRL, label.CAD, label.CHF, label.CLP, label.CNY, label.COP,
	label.CZK, label.DKK, label.DZD, label.EUR, label.HUF, label.INR, label.JPY, label.KWD, label.MAD, label.MXN,
	label.NGN, label.NOK, label.OMR, label.PLN, label.RSD, label.SAR, label.SDG, label.SEK, label.SGD, label.THB,
	label.TND, label.TRY, label.ZMW,
}

type fetcher struct {
	u *url.URL
	httputil.SourceHTTPClient
}

var _ provider.Source = (*source)(nil)

func NewSource(client *http.Client) *source {
	return &source{
		client: fetcher{
			u: &url.URL{
				Scheme: "https",
				Host:   hostname,
				Path:   "en/fx-rates",
			},
			SourceHTTPClient: httputil.NewHTTPClient(client),
		},
	}
}

type source struct {
	client fetcher
}

func (s *source) GetExchangeable() []label.Symbol {
	return exchangeableSymbols
}

func (s *source) FetchLatest(ctx context.Context) ([]provider.ExchangeRate, error) {
	list, err := s.fetchingPlan(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching plan: %w", err)
	}

	return list, nil
}

func (s *source) fetchingPlan(ctx context.Context) ([]provider.ExchangeRate, error) {
	currDate := time.Now().UTC().Format("02/01/2006")
	query := s.client.u.Query()
	query.Add("date_req", currDate)
	s.client.u.RawQuery = query.Encode()

	b, err := s.client.Get(ctx, *s.client.u)
	if err != nil {
		return nil, fmt.Errorf("fetching: %w", err)
	}

	list, err := s.decode(b)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return list, nil
}

func (s *source) decode(b []byte) ([]provider.ExchangeRate, error) {
	var list []provider.ExchangeRate

	aedSymRates := map[label.Symbol]float64{
		label.AED: 1,
	}

	aedExchangeRates, err := parseHTML(b)
	if err != nil {
		return nil, fmt.Errorf("decode xml: %w", err)
	}

	for _, r := range aedExchangeRates.rates {
		aedSymRates[r.symbol] = 1 / r.rate
	}

	aedExchangeRates.rates = append(
		aedExchangeRates.rates,
		aedExchangeRate{symbol: label.AED, rate: 1},
	)

	for _, sym := range aedExchangeRates.rates {
		for _, sym1 := range aedExchangeRates.rates {
			if sym.symbol != sym1.symbol {
				ccy, ok := label.Currencies[sym.symbol]
				if !ok {
					continue
				}

				ccy1, ok := label.Currencies[sym1.symbol]
				if !ok {
					continue
				}

				rate := ExchangeRate{
					time: aedExchangeRates.time,
					from: ccy,
					to:   ccy1,
					rate: aedSymRates[ccy1.Symbol] / aedSymRates[ccy.Symbol],
				}

				list = append(list, rate)
			}
		}
	}

	return list, nil
}
