package ecb

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

var testXMLHandlerFunc = func(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`
						<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
							<gesmes:subject>Reference rates</gesmes:subject>
							<gesmes:Sender>
								<gesmes:name>European Central Bank</gesmes:name>
							</gesmes:Sender>
							<Cube>
								<Cube time="2021-06-18">
									<Cube currency="USD" rate="1.1898"/>
									<Cube currency="JPY" rate="131.12"/>
								</Cube>
							</Cube>
						</gesmes:Envelope>
				`))
	w.Header().Set("Content-Type", "text/xml")
}

var testCsvHandlerFunc = func(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Date, USD, JPY\n 18 June 2021, 1.1898, 131.12"))
	w.Header().Set("Content-Type", "application/zip")
}

const (
	testXMLLatestPattern = "/latest/xml"
	testCSVLatestPattern = "/latest/csv"
)

func parseDatetime(datetime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", datetime)
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
		expected: 33,
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
		csvPattern     string
		xmlPattern     string
		handlerFunc    func() http.Handler
		requestTimeout time.Duration
	}{
		{
			name:           "fetch_latest_data_matching_" + label.USD.String() + "_" + label.JPY.String() + "_" + label.EUR.String(),
			datetime:       "2021-06-18",
			xmlPattern:     testXMLLatestPattern,
			csvPattern:     testCSVLatestPattern,
			requestTimeout: 10 * time.Second,
			data: []struct {
				from label.Currency
				to   label.Currency
				rate float64
			}{
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.JPY],
					rate: 110.203395528660279,
				},
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.EUR],
					rate: 0.8404773911581779,
				},
				{
					from: label.Currencies[label.JPY],
					to:   label.Currencies[label.USD],
					rate: 0.009074130567419158,
				},
				{
					from: label.Currencies[label.JPY],
					to:   label.Currencies[label.EUR],
					rate: 0.00762660158633313,
				},
				{
					from: label.Currencies[label.EUR],
					to:   label.Currencies[label.USD],
					rate: 1.1898,
				},
				{
					from: label.Currencies[label.EUR],
					to:   label.Currencies[label.JPY],
					rate: 131.12,
				},
			},
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(testXMLLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					testXMLHandlerFunc(w, r)
				})

				mux.HandleFunc(testCSVLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					testCsvHandlerFunc(w, r)
				})

				return mux
			},
		},
		{
			name:           "fetch_latest_large_request_timeout",
			datetime:       "2021-06-18",
			xmlPattern:     testXMLLatestPattern,
			csvPattern:     testCSVLatestPattern,
			err:            context.DeadlineExceeded,
			requestTimeout: 1 * time.Nanosecond,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(testXMLLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					testXMLHandlerFunc(w, r)
				})

				mux.HandleFunc(testCSVLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					testCsvHandlerFunc(w, r)
				})

				return mux
			},
		},
		{
			name:       "fetch_latest_decode_token_err",
			datetime:   "2021-06-18",
			xmlPattern: testXMLLatestPattern,
			err:        errDecodeToken,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(testXMLLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`
						<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
							<gesmes:subject>Reference rates</gesmes:subject>
							<gesmes:Sender>
								<gesmes:name>European Central Bank</gesmes:name>
							</gesmes:Sender>
							<Cube>
								<Cube time="2021-06-18">
									<Cube currency="USD" rate="1.1898"/>
									<Cube currency="JPY" rate="131.12"/>
								</Cube>
							</Cube>
						/gesmes:Envelope>
				`))
					w.Header().Set("Content-Type", "text/xml")
				})

				return mux
			},
		},
		{
			name:       "fetch_latest_http_not_ok",
			datetime:   "2021-06-18",
			xmlPattern: testXMLLatestPattern,
			err:        httputil.ErrStatusCode,
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(testXMLLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})

				mux.HandleFunc(testCSVLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})

				return mux
			},
		},
		{
			name:       "fetch_latest_body_gzip",
			datetime:   "2021-06-18",
			xmlPattern: testXMLLatestPattern,
			csvPattern: testCSVLatestPattern,
			data: []struct {
				from label.Currency
				to   label.Currency
				rate float64
			}{
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.JPY],
					rate: 110.203395528660279,
				},
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.EUR],
					rate: 0.8404773911581779,
				},
				{
					from: label.Currencies[label.JPY],
					to:   label.Currencies[label.USD],
					rate: 0.009074130567419158,
				},
				{
					from: label.Currencies[label.JPY],
					to:   label.Currencies[label.EUR],
					rate: 0.00762660158633313,
				},
				{
					from: label.Currencies[label.EUR],
					to:   label.Currencies[label.USD],
					rate: 1.1898,
				},
				{
					from: label.Currencies[label.EUR],
					to:   label.Currencies[label.JPY],
					rate: 131.12,
				},
			},
			handlerFunc: func() http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc(testXMLLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/x-gzip")
					b := make([]byte, 0)
					buf := bytes.NewBuffer(b)
					gz := gzip.NewWriter(buf)
					_, _ = gz.Write([]byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
							<gesmes:subject>Reference rates</gesmes:subject>
							<gesmes:Sender>
								<gesmes:name>European Central Bank</gesmes:name>
							</gesmes:Sender>
							<Cube>
								<Cube time="2021-06-18">
									<Cube currency="USD" rate="1.1898"/>
									<Cube currency="JPY" rate="131.12"/>
								</Cube>
							</Cube>
						</gesmes:Envelope>`))
					_ = gz.Flush()
					_, _ = w.Write(buf.Bytes())
				})

				mux.HandleFunc(testCSVLatestPattern, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/x-gzip")
					b := make([]byte, 0)
					buf := bytes.NewBuffer(b)
					gz := gzip.NewWriter(buf)
					_, _ = gz.Write([]byte("Date, USD, JPY\n 18 June 2021, 1.1898, 131.12"))
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

			fetchers := make([]fetcher, 0)

			if tc.xmlPattern != "" {
				xmlURL, err := url.Parse(srv.URL + tc.xmlPattern)
				if err != nil {
					t.Fatalf("unable to parse xml url: %v", err)
				}

				fetchers = append(fetchers, fetcher{
					latestURL:        *xmlURL,
					decodeFunc:       decodeXML(),
					SourceHTTPClient: httputil.NewHTTPClient(client),
				})
			}

			if tc.csvPattern != "" {
				csvURL, err := url.Parse(srv.URL + tc.csvPattern)
				if err != nil {
					t.Fatalf("unable to parse csv url: %v", err)
				}

				fetchers = append(fetchers, fetcher{
					latestURL:        *csvURL,
					decodeFunc:       decodeCSV(),
					SourceHTTPClient: httputil.NewHTTPClient(client),
				})
			}

			source := NewSource(client)
			source.fetchers = make([]fetcher, 0)
			source.fetchers = append(source.fetchers, fetchers...)

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
							t.Errorf("test %s-%s, mismatch (-want, +got):\n%s", datum.from.Symbol,
								datum.to.Symbol, diff,
							)
						}
					}
				}
			}
		})
	}
}
