package gokuu

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider"
)

func TestExchanger_RegisterProvider(t *testing.T) {
	t.Parallel()

	e := New(http.DefaultClient)
	tc := struct {
		expectedLen  int
		expectedName string
	}{
		expectedLen:  len(e.providers) + 1,
		expectedName: "TestName",
	}

	ctrl := gomock.NewController(t)
	source := provider.NewMockSource(ctrl)
	source.EXPECT().GetExchangeable().Return([]label.Symbol{})

	e.Register(tc.expectedName, source, 1)

	if diff := cmp.Diff(tc.expectedLen, len(e.providers)); diff != "" {
		t.Errorf("bad expected len (-want, +got): %s", diff)
	}

	for _, source := range e.providers {
		if source.name == tc.expectedName {
			return
		}
	}

	t.Errorf("unable find source with name: %s", tc.expectedName)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		excluded []string
	}{
		{
			name:     "test_without_ecb_cae",
			excluded: []string{ProviderNameECB, ProviderNameCAE},
		},
		{
			name:     "test_without_nil",
			excluded: nil,
		},
		{
			name:     "test_without_all",
			excluded: []string{ProviderNameCAE, ProviderNameECB, ProviderNameRCB},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := New(http.DefaultClient)
			e.Delete(tc.excluded...)
			for _, source := range e.providers {
				for _, s := range tc.excluded {
					if source.name == s {
						t.Errorf("found source with name: %s", source.name)
					}
				}
			}
		})
	}
}

func TestExchanger_GetExchangeable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		sourceSymbolsSetOne []label.Symbol
		sourceSymbolsSetTwo []label.Symbol
		expected            int
	}{
		{
			name:                "test_sources_diff",
			sourceSymbolsSetOne: []label.Symbol{label.RUB, label.USD},
			sourceSymbolsSetTwo: []label.Symbol{label.EUR, label.DKK},
			expected:            4,
		},
		{
			name:                "test_sources_identity",
			sourceSymbolsSetOne: []label.Symbol{label.RUB, label.USD},
			sourceSymbolsSetTwo: []label.Symbol{label.RUB, label.USD},
			expected:            2,
		},
		{
			name:                "test_sources_intersect",
			sourceSymbolsSetOne: []label.Symbol{label.RUB, label.USD},
			sourceSymbolsSetTwo: []label.Symbol{label.RUB, label.GBP},
			expected:            3,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sourceSetOne := provider.NewMockSource(ctrl)
			sourceSetOne.EXPECT().GetExchangeable().Return(tc.sourceSymbolsSetOne).AnyTimes()

			sourceSetTwo := provider.NewMockSource(ctrl)
			sourceSetTwo.EXPECT().GetExchangeable().Return(tc.sourceSymbolsSetTwo).AnyTimes()

			e := New(http.DefaultClient)
			e.providers = make([]*Provider, 0)
			e.providers = append(
				e.providers,
				&Provider{
					name:   "TestSourceOne",
					prior:  1,
					Source: sourceSetOne,
				},
				&Provider{
					name:   "TestSourceTwo",
					prior:  1,
					Source: sourceSetTwo,
				},
			)

			exchangeable := e.GetExchangeable()
			if diff := cmp.Diff(tc.expected, len(exchangeable)); diff != "" {
				t.Errorf("bad expected (-want, +got): %s", diff)
			}
		})
	}
}

func TestLatestResponse_Verify(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		expected       bool
		result         []ExchangeRate
		expectedResult []label.Symbol
	}{
		{
			name:           "test_equal",
			expected:       true,
			result:         []ExchangeRate{{}, {}, {}},
			expectedResult: []label.Symbol{label.GBP, label.USD, label.EUR},
		},
		{
			name:           "test_not_equal",
			expected:       false,
			result:         []ExchangeRate{{}, {}},
			expectedResult: []label.Symbol{label.GBP, label.USD, label.EUR},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := LatestResponse{Result: tc.result, Expected: tc.expectedResult}
			if diff := cmp.Diff(tc.expected, resp.Verify()); diff != "" {
				t.Errorf("bad expected (-want, +got): %s", diff)
			}
		})
	}
}

func TestExchanger_GetLatest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		mergeFunc   MergeFunc
		mergeTyp    MergeStrategyType
		labels      []label.Symbol
		expectedLen int
		expected    []struct {
			from, to label.Symbol
			rate     float64
		}
		sources []struct {
			name   string
			prior  Prior
			output []struct {
				from, to label.Symbol
				rate     float64
			}
		}
	}{
		{
			name:        "test_latest_merge_race_0",
			mergeFunc:   mergeRaceFunc(),
			mergeTyp:    MergeStrategyTypeRace,
			expectedLen: 2,
			labels:      []label.Symbol{label.GBP, label.USD},

			expected: []struct {
				from, to label.Symbol
				rate     float64
			}{
				{
					from: label.GBP,
					to:   label.USD,
					rate: 1.2,
				},
				{
					from: label.USD,
					to:   label.GBP,
					rate: 0.8,
				},
			},
			sources: []struct {
				name   string
				prior  Prior
				output []struct {
					from, to label.Symbol
					rate     float64
				}
			}{
				{
					name:  "test_source_0",
					prior: 0,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.GBP,
							to:   label.USD,
							rate: 1.2,
						},
						{
							from: label.USD,
							to:   label.GBP,
							rate: 0.8,
						},
					},
				},
			},
		},
		{
			name:        "test_latest_merge_race_1",
			mergeFunc:   mergeRaceFunc(),
			mergeTyp:    MergeStrategyTypeRace,
			expectedLen: 1,
			labels:      []label.Symbol{label.USD, label.USD},
			expected: []struct {
				from, to label.Symbol
				rate     float64
			}{
				{
					from: label.USD,
					to:   label.USD,
					rate: 1.2,
				},
			},
			sources: []struct {
				name   string
				prior  Prior
				output []struct {
					from, to label.Symbol
					rate     float64
				}
			}{
				{
					name:  "test_source_0",
					prior: 0,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.USD,
							to:   label.USD,
							rate: 1.2,
						},
					},
				},
			},
		},
		{
			name:        "test_latest_merge_priority_0",
			mergeFunc:   mergePriorFunc(),
			mergeTyp:    MergeStrategyTypePriority,
			expectedLen: 2,
			labels:      []label.Symbol{label.USD, label.GBP},
			expected: []struct {
				from, to label.Symbol
				rate     float64
			}{
				{
					from: label.GBP,
					to:   label.USD,
					rate: 3,
				},
				{
					from: label.USD,
					to:   label.GBP,
					rate: 0.3,
				},
			},
			sources: []struct {
				name   string
				prior  Prior
				output []struct {
					from, to label.Symbol
					rate     float64
				}
			}{
				{
					name:  "test_source_0",
					prior: 1,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.GBP,
							to:   label.USD,
							rate: 3,
						},
						{
							from: label.USD,
							to:   label.GBP,
							rate: 0.3,
						},
					},
				},
				{
					name:  "test_source_1",
					prior: 0,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.GBP,
							to:   label.USD,
							rate: 1.2,
						},
						{
							from: label.USD,
							to:   label.GBP,
							rate: 0.8,
						},
					},
				},
			},
		},
		{
			name:        "test_latest_merge_average_0",
			mergeFunc:   mergeAverageFunc(),
			mergeTyp:    MergeStrategyTypeAverage,
			expectedLen: 2,
			labels:      []label.Symbol{label.USD, label.GBP},
			expected: []struct {
				from, to label.Symbol
				rate     float64
			}{
				{
					from: label.GBP,
					to:   label.USD,
					rate: 5,
				},
				{
					from: label.USD,
					to:   label.GBP,
					rate: 2,
				},
			},
			sources: []struct {
				name   string
				prior  Prior
				output []struct {
					from, to label.Symbol
					rate     float64
				}
			}{
				{
					name:  "test_source_0",
					prior: 1,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.GBP,
							to:   label.USD,
							rate: 6,
						},
						{
							from: label.USD,
							to:   label.GBP,
							rate: 3,
						},
					},
				},
				{
					name:  "test_source_1",
					prior: 0,
					output: []struct {
						from, to label.Symbol
						rate     float64
					}{
						{
							from: label.GBP,
							to:   label.USD,
							rate: 4,
						},
						{
							from: label.USD,
							to:   label.GBP,
							rate: 1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			opts := make([]Option, 0)

			switch tc.mergeTyp {
			case MergeStrategyTypePriority:
				opts = append(opts, WithPriorityMergeStrategy())
			case MergeStrategyTypeRace:
				opts = append(opts, WithRaceMergeStrategy())
			case MergeStrategyTypeAverage:
				opts = append(opts, WithAverageMergeStrategy())
			}

			e := New(http.DefaultClient, opts...)
			e.merger = tc.mergeFunc
			e.providers = make([]*Provider, 0)
			for _, s := range tc.sources {
				var rates []provider.ExchangeRate
				for idx, set := range s.output {
					fromCcy, ok := label.Currencies[set.from]
					if !ok {
						t.Fatalf("unable find %s in label.Currencies", set.from)
					}

					toCcy, ok := label.Currencies[set.to]
					if !ok {
						t.Fatalf("unable find %s in label.Currencies", set.to)
					}

					rate := provider.NewMockExchangeRate(ctrl)
					rate.EXPECT().From().Return(fromCcy).AnyTimes()
					rate.EXPECT().To().Return(toCcy).AnyTimes()
					rate.EXPECT().Rate().Return(tc.expected[idx].rate).AnyTimes()
					rate.EXPECT().Time().Return(time.Now()).AnyTimes()

					rates = append(rates, rate)
				}

				source := provider.NewMockSource(ctrl)
				source.EXPECT().GetExchangeable().Return(tc.labels).AnyTimes()
				source.EXPECT().FetchLatest(gomock.Any()).Return(rates, nil).AnyTimes()

				e.Register(s.name, source, s.prior)
			}

			resp := e.GetLatest(ctx)

			if diff := cmp.Diff(tc.expectedLen, len(resp.Expected)); diff != "" {
				t.Errorf("bad expected (-want, +got): %s", diff)
			}

			if diff := cmp.Diff(0, len(resp.Unreceived)); diff != "" {
				t.Errorf("bad expected (-want, +got): %s", diff)
			}

		OuterLoop:
			for _, s := range tc.expected {
				for _, rate := range resp.Result {
					if s.from == rate.From().Symbol && s.to == rate.To().Symbol && s.rate == rate.Rate() {
						continue OuterLoop
					}
				}

				t.Errorf("exchange rate not found in response result")
			}
		})
	}
}
