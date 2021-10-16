package ecb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider"
	"github.com/robotomize/gokuu/provider/httputil"
)

const hostname = "www.ecb.europa.eu"

const (
	latestXMLRawPath = "/stats/eurofxref/eurofxref-daily.xml"
	latestCSVRawPath = "/stats/eurofxref/eurofxref.zip"
)

var (
	defaultLatestResourceCSV = url.URL{Scheme: "https", Host: hostname, Path: latestCSVRawPath}
	defaultLatestResourceXML = url.URL{Scheme: "https", Host: hostname, Path: latestXMLRawPath}
)

var exchangeableSymbols = []label.Symbol{
	label.USD, label.EUR, label.JPY, label.BGN, label.CZK, label.DKK, label.GBP, label.HUF, label.PLN,
	label.RON, label.SEK, label.CHF, label.ISK, label.NOK, label.HRK, label.RUB, label.TRY, label.AUD,
	label.BRL, label.CAD, label.CNY, label.HKD, label.IDR, label.ILS, label.INR, label.KRW, label.MXN, label.MYR,
	label.NZD, label.PHP, label.SGD, label.THB, label.ZAR,
}

var _ provider.Source = (*source)(nil)

type fetcher struct {
	latestURL url.URL
	decodeFunc
	httputil.SourceHTTPClient
}

func NewSource(client *http.Client) *source {
	httpClient := httputil.NewHTTPClient(client)

	return &source{
		fetchers: []fetcher{{
			latestURL:        defaultLatestResourceCSV,
			decodeFunc:       decodeCSV(),
			SourceHTTPClient: httpClient,
		}, {
			latestURL:        defaultLatestResourceXML,
			decodeFunc:       decodeXML(),
			SourceHTTPClient: httpClient,
		}},
	}
}

type source struct {
	fetchers []fetcher
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
	type fetchingDat struct {
		err error
		b   []byte
		d   decodeFunc
	}

	var dat fetchingDat
	var ferr *multierror.Error

	wg := sync.WaitGroup{}
	wg.Add(1)

	ch := make(chan fetchingDat)
	stopCh := make(chan struct{})

	for _, fet := range s.fetchers {
		fet := fet
		go func() {
			select {
			case <-stopCh:
				return
			default:
			}

			b, err := fet.Get(ctx, fet.latestURL)

			select {
			case <-stopCh:
				return
			case ch <- fetchingDat{b: b, d: fet.decodeFunc, err: err}:
			}
		}()
	}

	go func() {
		defer wg.Done()
		defer close(stopCh)
		n := len(s.fetchers)
		for {
			select {
			case <-ctx.Done():
				ferr = multierror.Append(ferr, fmt.Errorf("ctx cancelled: %w", ctx.Err()))
				return
			case dat = <-ch:
				n--
				if dat.err == nil {
					return
				}
				ferr = multierror.Append(ferr, dat.err)
				if n == 0 {
					return
				}
			}
		}
	}()

	wg.Wait()

	if dat.b == nil || dat.d == nil {
		return nil, ferr.ErrorOrNil()
	}

	d, b := dat.d, dat.b
	list, err := s.decode(b, d)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return list, nil
}

func (s *source) decode(b []byte, decodeFunc decodeFunc) ([]provider.ExchangeRate, error) {
	var list []provider.ExchangeRate

	euroSymRates := map[label.Symbol]float64{
		label.EUR: 1,
	}

	if err := decodeFunc(b, func(r euroLatestRates) error {
		for _, pair := range r.rates {
			euroSymRates[pair.symbol] = pair.rate
		}

		r.rates = append(r.rates, euroExchangeRate{symbol: label.EUR, rate: 1})

		for _, sym := range r.rates {
			for _, sym1 := range r.rates {
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
						time: r.time,
						from: ccy,
						to:   ccy1,
						rate: euroSymRates[ccy1.Symbol] / euroSymRates[ccy.Symbol],
					}

					list = append(list, rate)
				}
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("%T decode func: %w", decodeFunc, err)
	}

	return list, nil
}
