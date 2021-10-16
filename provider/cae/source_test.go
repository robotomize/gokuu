package cae

import (
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
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Date12-08-2021</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">12 Aug 2021 6:00PM</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>US Dollar</td>
                        <td>3.672500</td>
                    </tr>
                   <tr>
                        <td>Australian Dollar</td>
                        <td>2.696006</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`))
	w.Header().Set("Content-Type", "text/html")
}

var strPattern = "/rates"

func TestSource_GetExchangeable(t *testing.T) {
	t.Parallel()

	tc := struct {
		name     string
		expected int
	}{
		name:     "test_source_get_exchangeable",
		expected: 34,
	}

	client := http.DefaultClient
	source := NewSource(client)

	if tc.expected != len(source.GetExchangeable()) {
		diff := cmp.Diff(tc.expected, len(source.GetExchangeable()))
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestSource_fetchLatest(t *testing.T) {
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
			name:           "fetch_latest_data_matching_" + label.AUD.String() + label.AED.String() + label.USD.String(),
			datetime:       "12-08-2021",
			requestTimeout: 10 * time.Second,
			data: []struct {
				from label.Currency
				to   label.Currency
				rate float64
			}{
				{
					from: label.Currencies[label.AED],
					to:   label.Currencies[label.USD],
					rate: 0.27229407760381213,
				},
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.AED],
					rate: 3.6725,
				},
				{
					from: label.Currencies[label.USD],
					to:   label.Currencies[label.AUD],
					rate: 1.362200232492064,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.USD],
					rate: 0.7341064669843432,
				},
				{
					from: label.Currencies[label.AED],
					to:   label.Currencies[label.AUD],
					rate: 0.370919055818125,
				},
				{
					from: label.Currencies[label.AUD],
					to:   label.Currencies[label.AED],
					rate: 2.6960060000000002,
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
				if r.Time() != datetime {
					diff := cmp.Diff(datetime, r.Time())
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				for _, datum := range tc.data {
					if datum.from == r.From() && datum.to == r.To() {
						if diff := cmp.Diff(datum.rate, r.Rate()); diff != "" {
							t.Errorf("test %s-%s, mismatch (-want, +got):\n%s", datum.from.Symbol, datum.to.Symbol,
								diff,
							)
						}
					}
				}
			}
		})
	}
}

func parseDatetime(datetime string) (time.Time, error) {
	t, err := time.Parse("02-01-2006", datetime)
	if err != nil {
		return time.Time{}, fmt.Errorf("time parse: %w", err)
	}

	return t, nil
}
