package ecb

import (
	"time"

	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider"
)

var _ provider.ExchangeRate = (*ExchangeRate)(nil)

type ExchangeRate struct {
	time time.Time
	from label.Currency
	to   label.Currency
	rate float64
}

func (e ExchangeRate) Time() time.Time {
	return e.time
}

func (e ExchangeRate) From() label.Currency {
	return e.from
}

func (e ExchangeRate) To() label.Currency {
	return e.to
}

func (e ExchangeRate) Rate() float64 {
	return e.rate
}
