package rcb

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/robotomize/gokuu/label"
	"github.com/robotomize/gokuu/provider/httputil"
)

var handlerFunc = func(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<ValCurs Date="30.07.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>AUD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>AZN</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>GBP</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>102,1811</Value>
    </Valute>
</ValCurs>`))
	w.Header().Set("Content-Type", "text/xml")
}

var strPattern = "/rates"

func parseDatetime(datetime string) (time.Time, error) {
	t, err := time.Parse("02.01.2006", datetime)
	if err != nil {
		return time.Time{}, fmt.Errorf("time parse: %w", err)
	}

	return t, nil
}

func TestSource_GetExchangeable(t *testing.T) {
	t.Parallel()

	tc := struct {
		name     string
		expected int
	}{
		name:     "test_source_get_exchangeable",
		expected: 35,
	}

	client := http.DefaultClient
	source := NewSource(client)

	if diff := cmp.Diff(tc.expected, len(source.GetExchangeable())); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestSource_FetchLatest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		err      error
		datetime string
		data     []struct {
			from label.Currency
			to   label.Currency
			rate float64
		}
		handlerFunc    func() http.Handler
		requestTimeout time.Duration
	}{
		{
			name:           "fetch_latest_data_matching_" + label.AUD.String() + "_" + label.AZN.String() + "_" + label.GBP.String(),
			datetime:       "30.07.2021",
			requestTimeout: 10 * time.Second,
			data: []struct {
				from label.Currency
				to   label.Currency
				rate float64
			}{
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.AZN],
					rate: 1.2572605824251075,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.GBP],
					rate: 0.5300481204449746,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.RUB],
					rate: 54.1609,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.AUD],
					rate: 0.7953800620004469,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.GBP],
					rate: 0.4215897069027442,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.RUB],
					rate: 43.0785,
				},
			},
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(strPattern, func(w http.ResponseWriter, r *http.Request) {
					handlerFunc(w, r)
				})

				return mux
			},
		},
		{
			name:           "fetch_latest_large_request_timeout",
			datetime:       "2021-06-18",
			err:            context.DeadlineExceeded,
			requestTimeout: 1 * time.Nanosecond,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(strPattern, func(w http.ResponseWriter, r *http.Request) {
					handlerFunc(w, r)
				})

				return mux
			},
		},
		{
			name:     "fetch_latest_decode_token_err",
			datetime: "2021-06-18",
			err:      errDecodeToken,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(strPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`
						<ValCurs Date="30.07.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>AUD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>AZN</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>GBP</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>102,1811</Value>
    </Valute>
ValCurs>`))
					w.Header().Set("Content-Type", "text/xml")
				})

				return mux
			},
		},
		{
			name:     "fetch_latest_http_not_ok",
			datetime: "2021-06-18",
			err:      httputil.ErrStatusCode,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(strPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})

				return mux
			},
		},
		{
			name:     "fetch_latest_body_gzip",
			datetime: "30.07.2021",
			data: []struct {
				from label.Currency
				to   label.Currency
				rate float64
			}{
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.AZN],
					rate: 1.2572605824251075,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.GBP],
					rate: 0.5300481204449746,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.RUB],
					rate: 54.1609,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.AUD],
					rate: 0.7953800620004469,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.GBP],
					rate: 0.4215897069027442,
				},
				{
					from: label.Currencies[label.AZN],
					to:   label.Currencies[label.RUB],
					rate: 43.0785,
				},
			},
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(strPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/x-gzip")
					b := make([]byte, 0)
					buf := bytes.NewBuffer(b)
					gz := gzip.NewWriter(buf)
					_, _ = gz.Write([]byte(`<ValCurs Date="30.07.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>AUD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>AZN</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>GBP</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>102,1811</Value>
    </Valute>
</ValCurs>`))
					_ = gz.Flush()
					_, _ = w.Write(buf.Bytes())
				})

				return mux
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := tc.handlerFunc()
			srv := httptest.NewServer(h)
			client := srv.Client()

			source := NewSource(client)

			u, err := url.Parse(srv.URL + strPattern)
			if err != nil {
				t.Fatalf("unable to parse csv url: %v", err)
			}

			source.client.u = u

			var timeout time.Duration
			if tc.requestTimeout == 0 {
				timeout = 10 * time.Second
			} else {
				timeout = tc.requestTimeout
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			rates, err := source.FetchLatest(ctx)
			if tc.err != nil && err == nil {
				diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
				t.Errorf("mismatch (-want, +got):\n%s", diff)
				return
			}

			if err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("fetch latest rates: %v", err)
				}

				return
			}

			datetime, err := parseDatetime(tc.datetime)
			if err != nil {
				t.Errorf("datetime in test data invalid")
			}

			for _, r := range rates {
				if diff := cmp.Diff(datetime, r.Time()); diff != "" {
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				for _, datum := range tc.data {
					if datum.from == r.From() && datum.to == r.To() {
						if diff := cmp.Diff(datum.rate, r.Rate()); diff != "" {
							t.Errorf("test %s-%s, mismatch (-want, +got):\n%s",
								datum.from.Symbol, datum.to.Symbol, diff,
							)
						}
					}
				}
			}
		})
	}
}
