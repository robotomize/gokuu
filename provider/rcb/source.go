package rcb

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

const hostname = "cbr.ru"

var exchangeableSymbols = []label.Symbol{
	label.RUB, label.AUD, label.AZN, label.GBP, label.AMD, label.BYN, label.BGN, label.BRL, label.HUF, label.HKD, label.DKK, label.USD,
	label.EUR, label.INR, label.KZT, label.CAD, label.KGS, label.CNY, label.MDL, label.NOK, label.PLN, label.RON, label.XDR,
	label.SGD, label.TJS, label.TRY, label.TMT, label.UZS, label.UAH, label.CZK, label.SEK, label.CHF, label.ZAR, label.KRW,
	label.JPY,
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
				Path:   "scripts/XML_daily.asp",
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

	rubSymRates := map[label.Symbol]float64{
		label.RUB: 1,
	}

	rubExchangeRates, err := decodeXML(b)
	if err != nil {
		return nil, fmt.Errorf("decode xml: %w", err)
	}

	for _, r := range rubExchangeRates.rates {
		rubSymRates[r.symbol] = 1 / r.rate
	}

	rubExchangeRates.rates = append(
		rubExchangeRates.rates,
		rubExchangeRate{symbol: label.RUB, rate: 1},
	)

	for _, sym := range rubExchangeRates.rates {
		for _, sym1 := range rubExchangeRates.rates {
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
					time: rubExchangeRates.time,
					from: ccy,
					to:   ccy1,
					rate: rubSymRates[ccy1.Symbol] / rubSymRates[ccy.Symbol],
				}

				list = append(list, rate)
			}
		}
	}

	return list, nil
}
