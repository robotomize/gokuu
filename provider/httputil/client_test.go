package httputil

import (
	"net/http"
	"testing"
)

// Nice test check :)
func TestHTTPClient_UserAgent(t *testing.T) {
	t.Parallel()
	client := NewHTTPClient(http.DefaultClient)

	if client.UserAgent() != "gokuu/0.0.0" {
		t.Errorf("user agent wrong")
	}
}
