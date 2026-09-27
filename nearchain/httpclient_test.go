package nearchain

import (
	"net/http"
	"testing"
	"time"
)

// A dead pooled HTTP/2 connection must be detected by health pings, not reused until each request
// times out (it froze the batcher on testnet until a restart).
func TestHTTPClientPingsHTTP2Connections(t *testing.T) {
	c := NewHTTPClient(30 * time.Second)
	tr, ok := c.Transport.(*http.Transport)
	if !ok || tr.HTTP2 == nil || tr.HTTP2.SendPingTimeout <= 0 || tr.HTTP2.PingTimeout <= 0 || !tr.ForceAttemptHTTP2 {
		t.Fatalf("HTTP/2 health checks not configured: %+v", tr)
	}
	if NewClient("http://x").HTTP.Transport != nil && NewClient("http://x").HTTP.Transport.(*http.Transport).HTTP2 == nil {
		t.Fatal("the RPC client must use it")
	}
}
