package proxy

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	cryptoTLS "crypto/tls"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

type Rotator struct {
	db      *sql.DB
	mu      sync.RWMutex
	proxies []*url.URL
	index   int
}

// NewRotator creates a new Rotator instance and starts background reloading
func NewRotator(db *sql.DB) (*Rotator, error) {
	r := &Rotator{
		db: db,
	}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	// Start background reloader to fetch changes from DB periodically
	go r.startBackgroundReload(15 * time.Second)
	return r, nil
}

// Reload reads active proxies from the database and updates the in-memory pool
func (r *Rotator) Reload() error {
	rows, err := r.db.Query("SELECT url FROM proxies WHERE status = 'active'")
	if err != nil {
		return fmt.Errorf("failed to fetch proxies from db: %w", err)
	}
	defer rows.Close()

	var newProxies []*url.URL
	for rows.Next() {
		var uStr string
		if err := rows.Scan(&uStr); err != nil {
			return err
		}
		parsed, err := url.Parse(uStr)
		if err != nil {
			// Skip invalid URLs but keep reading others
			continue
		}
		newProxies = append(newProxies, parsed)
	}

	r.mu.Lock()
	r.proxies = newProxies
	if r.index >= len(newProxies) {
		r.index = 0
	}
	r.mu.Unlock()
	return nil
}

func (r *Rotator) startBackgroundReload(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		_ = r.Reload()
	}
}

// GetNextProxy returns the next proxy URL in the round-robin rotation with per-request session ID injection for IP rotation
func (r *Rotator) GetNextProxy() (*url.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.proxies) == 0 {
		return nil, nil // No active proxies, use direct connection
	}

	p := r.proxies[r.index]
	r.index = (r.index + 1) % len(r.proxies)

	// Make a shallow copy of the URL to safely modify the credentials without corrupting the cached value
	uCopy := *p
	if uCopy.User != nil {
		username := uCopy.User.Username()
		password, _ := uCopy.User.Password()
		
		// Only append a session identifier if not already present
		if !strings.Contains(username, "-session-") {
			randBytes := make([]byte, 8)
			_, _ = rand.Read(randBytes)
			sessionStr := fmt.Sprintf("%x", randBytes)
			username = fmt.Sprintf("%s-session-%s", username, sessionStr)
			uCopy.User = url.UserPassword(username, password)
		}
	}
	return &uCopy, nil
}

// RotatorRoundTripper is a custom RoundTripper that wraps Transport to rotate proxies
type RotatorRoundTripper struct {
	rotator     *Rotator
	h1Transport *http.Transport
	h2Transport *http2.Transport
}

// GetRoundTripper returns an http.RoundTripper using the rotator
func (r *Rotator) GetRoundTripper() http.RoundTripper {
	h2Transport := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *cryptoTLS.Config) (net.Conn, error) {
			proxyURL, err := r.GetNextProxy()
			if err != nil {
				return nil, err
			}
			return dialTLSWithUTLS(ctx, network, addr, proxyURL)
		},
	}

	h1Transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return r.GetNextProxy()
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &RotatorRoundTripper{
		rotator:     r,
		h1Transport: h1Transport,
		h2Transport: h2Transport,
	}
}

func (rt *RotatorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "https" {
		return rt.h2Transport.RoundTrip(req)
	}
	return rt.h1Transport.RoundTrip(req)
}

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func dialTLSWithUTLS(ctx context.Context, network, addr string, proxyURL *url.URL) (net.Conn, error) {
	var conn net.Conn
	var err error

	if proxyURL != nil {
		// 1. Dial the proxy server (HTTP proxy)
		dialer := &net.Dialer{}
		conn, err = dialer.DialContext(ctx, "tcp", proxyURL.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to dial proxy server %s: %w", proxyURL.Host, err)
		}

		// 2. Establish HTTP CONNECT tunnel to target host
		connectReq := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: addr},
			Host:   addr,
			Header: make(http.Header),
		}
		connectReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
		connectReq.Header.Set("Proxy-Connection", "Keep-Alive")

		if proxyURL.User != nil {
			username := proxyURL.User.Username()
			password, _ := proxyURL.User.Password()
			auth := username + ":" + password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			connectReq.Header.Set("Proxy-Authorization", basicAuth)
		}

		err = connectReq.Write(conn)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to write CONNECT request to proxy: %w", err)
		}

		// 3. Read response from proxy
		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, connectReq)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read response from proxy CONNECT: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			conn.Close()
			return nil, fmt.Errorf("proxy CONNECT tunnel failed with status: %s", resp.Status)
		}

		// Wrap connection to consume buffered bytes first
		conn = &bufferedConn{
			Conn: conn,
			r:    br,
		}
	} else {
		// Direct TCP connection if no proxy is configured
		dialer := &net.Dialer{}
		conn, err = dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial target %s: %w", addr, err)
		}
	}

	// 4. Perform TLS handshake using uTLS to mimic Chrome
	host, _, _ := net.SplitHostPort(addr)
	config := &utls.Config{
		ServerName: host,
		NextProtos: []string{"h2", "http/1.1"},
	}

	uconn := utls.UClient(conn, config, utls.HelloChrome_Auto)
	err = uconn.Handshake()
	if err != nil {
		uconn.Close()
		return nil, fmt.Errorf("uTLS handshake failed with target %s: %w", addr, err)
	}

	return &tlsConnWrapper{UConn: uconn}, nil
}

// tlsConnWrapper wraps *utls.UConn and maps ConnectionState() to return standard crypto/tls.ConnectionState
type tlsConnWrapper struct {
	*utls.UConn
}

func (w *tlsConnWrapper) ConnectionState() cryptoTLS.ConnectionState {
	cs := w.UConn.ConnectionState()
	return cryptoTLS.ConnectionState{
		Version:                    cs.Version,
		HandshakeComplete:          cs.HandshakeComplete,
		DidResume:                  cs.DidResume,
		CipherSuite:                cs.CipherSuite,
		NegotiatedProtocol:         cs.NegotiatedProtocol,
		NegotiatedProtocolIsMutual: cs.NegotiatedProtocolIsMutual,
		ServerName:                 cs.ServerName,
		PeerCertificates:          cs.PeerCertificates,
		VerifiedChains:             cs.VerifiedChains,
		SignedCertificateTimestamps: cs.SignedCertificateTimestamps,
		OCSPResponse:               cs.OCSPResponse,
		TLSUnique:                  cs.TLSUnique,
	}
}

// ParseProxyURL parses different formats of proxy configurations into a standard URL format
func ParseProxyURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty proxy string")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	}

	// Try splitting by colon to check if it's host:port:username:password
	parts := strings.Split(raw, ":")
	if len(parts) == 4 {
		host := parts[0]
		port := parts[1]
		user := parts[2]
		pass := parts[3]
		return fmt.Sprintf("http://%s:%s@%s:%s", user, pass, host, port), nil
	} else if len(parts) == 2 {
		// host:port
		return fmt.Sprintf("http://%s:%s", parts[0], parts[1]), nil
	}

	// Fallback to prepending http:// and parsing
	u, err := url.Parse("http://" + raw)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
