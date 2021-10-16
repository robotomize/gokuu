package rcb

import (
	"time"

	"github.com/robotomize/gokuu/label"
)

type rubLatestRates struct {
	time  time.Time
	rates []rubExchangeRate
}

type rubExchangeRate struct {
	symbol label.Symbol
	rate   float64
}
