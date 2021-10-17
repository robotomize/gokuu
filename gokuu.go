package gokuu

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider"
	"github.com/robotomize/gokuu/provider/cae"
	"github.com/robotomize/gokuu/provider/ecb"
	"github.com/robotomize/gokuu/provider/rcb"
	"github.com/sethvargo/go-retry"
)

var (
	ErrConversionRate   = errors.New("can not convert")
	ErrCurrencyNotFound = errors.New("currency symbol is not supported")
)

const (
	DefaultRequestTimeout = 10 * time.Second
	DefaultRetryNum       = 1
	DefaultRetryDuration  = 5 * time.Second
)

const (
	// ProviderNameECB source name for European central bank
	ProviderNameECB = "ecb"
	// ProviderNameRCB source name for the Russia central bank
	ProviderNameRCB = "rcb"
	// ProviderNameCAE source name for the UAE central bank
	ProviderNameCAE = "cae"
)

type Exchanger interface {
	GetExchangeable() []label.Symbol
	GetLatest() LatestResponse
	Convert(ctx context.Context, from, to label.Symbol, value float64) (ConversionResponse, error)
}

type Option func(*exchanger)

type Options struct {
	RetryNum       uint64
	RetryDuration  time.Duration
	RequestTimeout time.Duration
	MergeStrategy  MergeStrategyType
}

type LatestResponse struct {
	Expected   []label.Symbol
	Unreceived []label.Symbol
	Info       []SourceInfo
	Result     []ExchangeRate
}

func (e LatestResponse) Verify() bool {
	return len(e.Expected) == len(e.Result)
}

type ProviderRespStatus byte

const (
	ProviderRespStatusFailed ProviderRespStatus = iota
	ProviderRespStatusOK
)

type SourceInfo struct {
	Name         string
	Status       ProviderRespStatus
	ErrorMessage string
}

type MergeStrategyType string

const (
	MergeStrategyTypeRace     MergeStrategyType = "race"
	MergeStrategyTypeAverage  MergeStrategyType = "average"
	MergeStrategyTypePriority MergeStrategyType = "priority"
)

type Prior int32

type Provider struct {
	name  string
	prior Prior
	provider.Source
}

// WithAverageMergeStrategy use the merge strategy to calculate the average Value of exchange rates
// with a large number of providers
func WithAverageMergeStrategy() Option {
	return func(g *exchanger) {
		g.opts.MergeStrategy = MergeStrategyTypeAverage
	}
}

// WithRaceMergeStrategy use the "who's the fastest" merger strategy. Duplicate data that came later are discarded
func WithRaceMergeStrategy() Option {
	return func(g *exchanger) {
		g.opts.MergeStrategy = MergeStrategyTypeRace
	}
}

// WithPriorityMergeStrategy use a strategy of merging by priority. You can set the priority of each source.
// You'll get a response based on priorities
func WithPriorityMergeStrategy() Option {
	return func(g *exchanger) {
		g.opts.MergeStrategy = MergeStrategyTypePriority
	}
}

// WithMergeFunc set the custom currency merge function
func WithMergeFunc(f MergeFunc) Option {
	return func(g *exchanger) {
		g.merger = f
	}
}

// WithRetryNum set number of repeated requests for data retrieval errors from the source
func WithRetryNum(n uint64) Option {
	return func(e *exchanger) {
		e.opts.RetryNum = n
	}
}

// WithRetryDuration max retry backoff
func WithRetryDuration(t time.Duration) Option {
	return func(e *exchanger) {
		e.opts.RetryDuration = t
	}
}

// WithRequestTimeout set a timeout for source requests
func WithRequestTimeout(t time.Duration) Option {
	return func(e *exchanger) {
		e.opts.RequestTimeout = t
	}
}

// New return exchanger
func New(client *http.Client, opts ...Option) *exchanger {
	e := &exchanger{
		opts: Options{
			RetryNum:       DefaultRetryNum,
			RetryDuration:  DefaultRetryDuration,
			RequestTimeout: DefaultRequestTimeout,
			MergeStrategy:  MergeStrategyTypeRace,
		},
		providers: []*Provider{
			{
				name:   ProviderNameECB,
				prior:  0,
				Source: ecb.NewSource(client),
			},
			{
				name:   ProviderNameRCB,
				prior:  1,
				Source: rcb.NewSource(client),
			},
			{
				name:   ProviderNameCAE,
				prior:  2,
				Source: cae.NewSource(client),
			},
		},
	}

	for _, opt := range opts {
		opt(e)
	}

	sort.Slice(e.providers, func(i, j int) bool {
		return e.providers[i].prior > e.providers[j].prior
	})

	return e
}

type exchanger struct {
	opts Options

	mtx          sync.RWMutex
	providers    []*Provider
	exchangeable []label.Symbol
	merger       MergeFunc
}

type FetchFunc func(ctx context.Context) LatestResponse

type ConvOpt struct {
	From    label.Symbol
	To      label.Symbol
	Value   float64
	CacheFn FetchFunc
}

// Convert returns an object with currency conversion data.
// The CacheFn option allows you to define your own data delivery function for caching
//
//
// 	ctx := context.Background()
//	g := gokuu.New()
//	latest := g.GetLatest(ctx)
//	g.Convert(
//		ctx, gokuu.ConvOpt{
//			From:  label.EUR,
//			To:    label.USD,
//			Value: 10,
//			CacheFn: func(ctx context.Context) LatestResponse {
//				return latest
//			},
//		}
//	)
func (e *exchanger) Convert(ctx context.Context, param ConvOpt) (ConversionResponse, error) {
	var resp ConversionResponse

	e.verifyExchangeable()

	e.mtx.RLock()
	defer e.mtx.RUnlock()

	fromCurrency, ok := label.Currencies[param.From]
	if !ok {
		return resp, ErrCurrencyNotFound
	}

	toCurrency, ok := label.Currencies[param.To]
	if !ok {
		return resp, ErrCurrencyNotFound
	}

	res := e.isExchangeable(param.From, param.To)
	if !res.from {
		return resp, fmt.Errorf("%w: %s", ErrCurrencyNotFound, param.From)
	}

	if !res.to {
		return resp, fmt.Errorf("%w: %s", ErrCurrencyNotFound, param.To)
	}

	if param.CacheFn == nil {
		param.CacheFn = e.getLatest
	}

	latest := param.CacheFn(ctx)

	var r *ExchangeRate

	for _, rate := range latest.Result {
		rate := rate
		if rate.from.Symbol == param.From && rate.to.Symbol == param.To {
			r = &rate
			break
		}
	}

	if r == nil {
		return ConversionResponse{
			Value: param.Value,
			From:  fromCurrency,
			To:    toCurrency,
			Info:  latest.Info,
		}, ErrConversionRate
	}

	return ConversionResponse{
		Value:  param.Value,
		From:   r.from,
		To:     r.to,
		Rate:   r.rate,
		Amount: r.rate * param.Value,
		Info:   latest.Info,
	}, nil
}

// GetLatest returns the current exchange rate for multiple currencies
func (e *exchanger) GetLatest(ctx context.Context) LatestResponse {
	e.verifyExchangeable()

	e.mtx.RLock()
	defer e.mtx.RUnlock()

	return e.getLatest(ctx)
}

// GetExchangeable returns a list of all available exchange rates in gokuu
// If any data provider is unavailable, GetExchangeable will still display a list of all possible exchange rates
func (e *exchanger) GetExchangeable() []label.Symbol {
	e.verifyExchangeable()

	e.mtx.RLock()
	defer e.mtx.RUnlock()

	return e.exchangeable
}

// Delete providers by name
func (e *exchanger) Delete(names ...string) {
	for _, name := range names {
		for idx, source := range e.providers {
			if source.name == name {
				source = nil
				e.providers = append(e.providers[:idx], e.providers[idx+1:]...)
				break
			}
		}
	}
}

// ChangePrior change provider priority
func (e *exchanger) ChangePrior(name string, prior Prior) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	for _, p := range e.providers {
		if p.name == name {
			p.prior = prior
		}
	}

	sort.Slice(e.providers, func(i, j int) bool {
		return e.providers[i].prior > e.providers[j].prior
	})
}

// Register allows you to add your own provider of exchange rate data
func (e *exchanger) Register(name string, source provider.Source, prior Prior) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.providers = append(e.providers, &Provider{
		name:   name,
		Source: source,
		prior:  prior,
	})

	sort.Slice(e.providers, func(i, j int) bool {
		return e.providers[i].prior > e.providers[j].prior
	})

	e.updateExchangeable()
}

type ConversionResponse struct {
	Date   time.Time
	Value  float64
	From   label.Currency
	To     label.Currency
	Rate   float64
	Amount float64
	Info   []SourceInfo
}

func (e ConversionResponse) String() string {
	return fmt.Sprintf(
		"Value: %f, From: %s, To: %s, Rate: %f, Amount: %f",
		e.Value,
		e.From.Symbol,
		e.To.Symbol,
		e.Rate,
		e.Amount,
	)
}

var _ provider.ExchangeRate = (*ExchangeRate)(nil)

type ExchangeRate struct {
	priority Prior
	time     time.Time
	from     label.Currency
	to       label.Currency
	rate     float64
}

func (r ExchangeRate) Time() time.Time {
	return r.time
}

func (r ExchangeRate) From() label.Currency {
	return r.from
}

func (r ExchangeRate) To() label.Currency {
	return r.to
}

func (r ExchangeRate) Rate() float64 {
	return r.rate
}

type MergeFunc func(*BatchExchanges, []ExchangeRate)

func mergerFor(strategy MergeStrategyType) MergeFunc {
	var f MergeFunc
	switch strategy {
	case MergeStrategyTypeRace:
		f = mergeRaceFunc()
	case MergeStrategyTypeAverage:
		f = mergeAverageFunc()
	case MergeStrategyTypePriority:
		f = mergePriorFunc()
	default:
		f = mergeRaceFunc()
	}

	return f
}

func mergeRaceFunc() MergeFunc {
	return func(batch *BatchExchanges, rates []ExchangeRate) {
		batch.walk(rates, func(curr, next ExchangeRate) (ExchangeRate, error) {
			if curr.from.Symbol != "" {
				return curr, nil
			}

			if next.to.Symbol != "" {
				return next, nil
			}

			return ExchangeRate{}, errors.New("d1 and d2 equals nil")
		})
	}
}

// mergeAverageFunc calculates the average of two exchange rates from different suppliers
func mergeAverageFunc() MergeFunc {
	return func(batch *BatchExchanges, rates []ExchangeRate) {
		batch.walk(rates, func(curr, next ExchangeRate) (ExchangeRate, error) {
			if curr.from.Symbol == "" || next.to.Symbol == "" {
				return ExchangeRate{}, errors.New("d1 or d2 equals nil")
			}

			return ExchangeRate{
				priority: curr.priority,
				time:     curr.time,
				from:     curr.from,
				to:       curr.to,
				rate:     (curr.rate + next.rate) / 2,
			}, nil
		})
	}
}

func mergePriorFunc() MergeFunc {
	return func(batch *BatchExchanges, rates []ExchangeRate) {
		batch.walk(rates, func(curr, next ExchangeRate) (ExchangeRate, error) {
			if curr.priority < next.priority {
				return next, nil
			}

			return curr, nil
		})
	}
}

func (e *exchanger) merge(batch *BatchExchanges, rates []ExchangeRate) {
	batch.mtx.Lock()
	defer batch.mtx.Unlock()

	mergeStrategyFn := mergerFor(e.opts.MergeStrategy)

	mergeStrategyFn(batch, rates)
}

func (e *exchanger) expandRates(source *Provider, rates []provider.ExchangeRate) []ExchangeRate {
	list := make([]ExchangeRate, len(rates))
	for i := range rates {
		list[i] = ExchangeRate{
			priority: source.prior,
			time:     rates[i].Time(),
			from:     rates[i].From(),
			to:       rates[i].To(),
			rate:     rates[i].Rate(),
		}
	}

	return list
}

func (e *exchanger) getLatest(ctx context.Context) LatestResponse {
	var wg sync.WaitGroup
	var mtx sync.RWMutex

	ctx, cancel := context.WithTimeout(ctx, e.opts.RequestTimeout)
	defer cancel()

	batch := &BatchExchanges{}
	resp := LatestResponse{
		Expected:   make([]label.Symbol, len(e.exchangeable)),
		Unreceived: make([]label.Symbol, 0),
		Info:       make([]SourceInfo, 0),
		Result:     make([]ExchangeRate, 0),
	}

	for _, source := range e.providers {
		source := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			report := SourceInfo{Name: source.name}

			b, _ := retry.NewConstant(e.opts.RetryDuration)

			b = retry.WithMaxRetries(e.opts.RetryNum, b)

			if err := retry.Do(ctx, b, func(ctx context.Context) error {
				rates, err := source.FetchLatest(ctx)
				if err != nil {
					return retry.RetryableError(fmt.Errorf("fetch latest: %w", err))
				}

				report.Status = ProviderRespStatusOK
				expanded := e.expandRates(source, rates)
				e.merge(batch, expanded)

				return nil
			}); err != nil {
				report.ErrorMessage = err.Error()
				report.Status = ProviderRespStatusFailed
			}

			mtx.Lock()
			defer mtx.Unlock()

			resp.Info = append(resp.Info, report)
		}()
	}

	wg.Wait()

	copy(resp.Expected, e.exchangeable)

	for _, symbol := range e.exchangeable {
		if _, ok := batch.Items[symbol]; !ok {
			resp.Unreceived = append(resp.Unreceived, symbol)
		}
	}

	if batch.Items == nil {
		return resp
	}

	resp.Result = append(resp.Result, batch.flatten()...)

	return resp
}

func (e *exchanger) isExchangeable(from, to label.Symbol) struct {
	from bool
	to   bool
} {
	res := struct {
		from bool
		to   bool
	}{}

	for _, symbol := range e.exchangeable {
		if symbol == from {
			res.from = true
		}

		if symbol == to {
			res.to = true
		}

		if res.from && res.to {
			return res
		}
	}

	return res
}

func (e *exchanger) verifyExchangeable() {
	e.mtx.RLock()
	if len(e.exchangeable) > 0 {
		e.mtx.RUnlock()
		return
	}
	e.mtx.RUnlock()

	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.updateExchangeable()
}

func (e *exchanger) updateExchangeable() {
	uniqLabels := make(map[label.Symbol]struct{})

	for _, source := range e.providers {
		for _, symbol := range source.GetExchangeable() {
			if _, ok := uniqLabels[symbol]; !ok {
				uniqLabels[symbol] = struct{}{}
			}
		}
	}

	e.exchangeable = make([]label.Symbol, 0, len(uniqLabels))
	for symbol := range uniqLabels {
		e.exchangeable = append(e.exchangeable, symbol)
	}
}

type BatchExchanges struct {
	mtx   sync.RWMutex
	Items map[label.Symbol]map[label.Symbol]ExchangeRate
}

// The map generates currency rates and allows you to bypass the entire map of objects and apply the function
// of merging currency rates from different providers
func (b *BatchExchanges) walk(rates []ExchangeRate, fn func(cur, next ExchangeRate) (ExchangeRate, error)) {
	if b.Items == nil {
		b.Items = make(map[label.Symbol]map[label.Symbol]ExchangeRate)
	}
	// rates bypass
	for _, r := range rates {
		from, to := r.from.Symbol, r.to.Symbol
		if extMap, ok := b.Items[from]; ok {
			if b.Items[from] == nil {
				b.Items[from] = make(map[label.Symbol]ExchangeRate)
			}

			if curr, yes := extMap[to]; yes {
				// if item exist call merge strategy function
				next, err := fn(curr, r)
				if err != nil {
					continue
				}
				// replacing the current item with an item from the merge function
				extMap[to] = next
			} else {
				extMap[to] = r
			}
		} else {
			b.Items[from] = make(map[label.Symbol]ExchangeRate)
			b.Items[from][to] = r
		}
	}
}

// convert batch exchanges map to exchange rate list
func (b *BatchExchanges) flatten() []ExchangeRate {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	list := make([]ExchangeRate, 0, len(b.Items)*2)
	for _, mpr := range b.Items {
		for _, r := range mpr {
			list = append(list, r)
		}
	}

	return list
}
