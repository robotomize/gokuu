package cae

import (
	"time"

	"github.com/robotomize/gokuu/label"
)

type aedLatestRates struct {
	time  time.Time
	rates []aedExchangeRate
}

type aedExchangeRate struct {
	symbol label.Symbol
	rate   float64
}
