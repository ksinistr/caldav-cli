package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksinistr/caldav-cli/internal/config"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				ServerURL:          "https://baikal.example.com/dav.php/",
				Username:           "alice",
				Password:           "testpass",
				InsecureSkipVerify: false,
			},
			wantErr: false,
		},
		{
			name: "valid config with insecure skip verify",
			cfg: &config.Config{
				ServerURL:          "https://baikal.example.com/dav.php/",
				Username:           "alice",
				Password:           "testpass",
				InsecureSkipVerify: true,
			},
			wantErr: false,
		},
		{
			name: "valid config with empty password",
			cfg: &config.Config{
				ServerURL:          "https://baikal.example.com/dav.php/",
				Username:           "alice",
				Password:           "",
				InsecureSkipVerify: false,
			},
			wantErr: false,
		},
		{
			name: "invalid server URL - client creation succeeds but will fail at runtime",
			cfg: &config.Config{
				ServerURL:          "",
				Username:           "alice",
				Password:           "testpass",
				InsecureSkipVerify: false,
			},
			wantErr: false, // webdav.NewClient doesn't validate URL at creation time
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client == nil {
					t.Error("NewClient() returned nil client")
				}
				if client.ServerURL() != tt.cfg.ServerURL {
					t.Errorf("Client.ServerURL() = %v, want %v", client.ServerURL(), tt.cfg.ServerURL)
				}
				if client.HTTPClient() == nil {
					t.Error("Client.HTTPClient() returned nil")
				}
				if client.CalDAVClient() == nil {
					t.Error("Client.CalDAVClient() returned nil")
				}
				if client.WebDAVClient() == nil {
					t.Error("Client.WebDAVClient() returned nil")
				}
			}
		})
	}
}

func TestClient_Accessors(t *testing.T) {
	cfg := &config.Config{
		ServerURL: "https://baikal.example.com/dav.php/",
		Username:  "alice",
		Password:  "testpass",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Test ServerURL
	if got := client.ServerURL(); got != cfg.ServerURL {
		t.Errorf("Client.ServerURL() = %v, want %v", got, cfg.ServerURL)
	}

	// Test HTTPClient
	if got := client.HTTPClient(); got == nil {
		t.Error("Client.HTTPClient() returned nil")
	}

	// Test CalDAVClient
	if got := client.CalDAVClient(); got == nil {
		t.Error("Client.CalDAVClient() returned nil")
	}

	// Test WebDAVClient
	if got := client.WebDAVClient(); got == nil {
		t.Error("Client.WebDAVClient() returned nil")
	}
}

func TestFindCurrentUserPrincipal(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php":
			http.Redirect(w, r, ts.URL+"/dav.php/", http.StatusMovedPermanently)
		case r.Method == "PROPFIND" && r.URL.Path == "/dav.php/":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response>
<D:href>/dav.php/</D:href>
<D:propstat>
<D:prop>
<D:current-user-principal>
<D:href>/dav.php/principals/alice/</D:href>
</D:current-user-principal>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`)
		case r.Method == http.MethodGet && r.URL.Path == "/dav.php/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "<html><body>sabre/dav</body></html>")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name      string
		serverURL string
	}{
		{
			name:      "with trailing slash",
			serverURL: ts.URL + "/dav.php/",
		},
		{
			name:      "without trailing slash",
			serverURL: ts.URL + "/dav.php",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(&config.Config{
				ServerURL: tt.serverURL,
				Username:  "alice",
				Password:  "testpass",
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			got, err := client.FindCurrentUserPrincipal(context.Background())
			if err != nil {
				t.Fatalf("FindCurrentUserPrincipal() error = %v", err)
			}
			if got != "/dav.php/principals/alice/" {
				t.Fatalf("FindCurrentUserPrincipal() = %q, want %q", got, "/dav.php/principals/alice/")
			}
		})
	}
}
