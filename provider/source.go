package provider

import (
	"context"
	"time"

	"github.com/robotomize/gokuu/label"
)

// Source is an interface for getting data from external sources. Source takes care of receiving data,
// working with proxies and giving back exchange rates
//
//go:generate mockgen -source source.go -destination mock_source.go -package provider
type Source interface {
	// FetchLatest  of obtaining the latest exchange rate data
	FetchLatest(ctx context.Context) ([]ExchangeRate, error)

	// GetExchangeable declares to give a list of exchangeable currencies
	GetExchangeable() []label.Symbol
}

// ExchangeRate represents the exchange rate of a particular currency pair
type ExchangeRate interface {
	// Time - date on which the exchange rate was issued
	Time() time.Time
	// From USD to EUR => 1USD ~ 1.17EUR
	From() label.Currency
	To() label.Currency
	Rate() float64
}
