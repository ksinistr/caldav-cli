package caldav

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/ksinistr/caldav-cli/internal/config"
)

type currentUserPrincipalPropFind struct {
	XMLName xml.Name                      `xml:"DAV: propfind"`
	Prop    currentUserPrincipalPropField `xml:"prop"`
}

type currentUserPrincipalPropField struct {
	CurrentUserPrincipal struct{} `xml:"DAV: current-user-principal"`
}

type currentUserPrincipalMultiStatus struct {
	XMLName   xml.Name                       `xml:"DAV: multistatus"`
	Responses []currentUserPrincipalResponse `xml:"response"`
}

type currentUserPrincipalResponse struct {
	PropStats []currentUserPrincipalPropStat `xml:"propstat"`
}

type currentUserPrincipalPropStat struct {
	Prop currentUserPrincipalPropValue `xml:"prop"`
}

type currentUserPrincipalPropValue struct {
	CurrentUserPrincipal currentUserPrincipalValue `xml:"DAV: current-user-principal"`
}

type currentUserPrincipalValue struct {
	Href            string    `xml:"href"`
	Unauthenticated *struct{} `xml:"unauthenticated"`
}

type multiStatusError struct {
	Status string
}

func (e *multiStatusError) Error() string {
	return fmt.Sprintf("HTTP multi-status request failed: %s", e.Status)
}

// Client wraps the CalDAV and WebDAV clients with Baikal-specific configuration.
type Client struct {
	http   *http.Client
	webdav *webdav.Client
	caldav *caldav.Client
	server *config.Config
}

// NewClient creates a new CalDAV client from the given configuration.
// It uses basic authentication and configures TLS settings based on the config.
func NewClient(cfg *config.Config) (*Client, error) {
	// Create HTTP client with basic auth and timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	if cfg.InsecureSkipVerify {
		// Create a custom transport that skips TLS verification
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Wrap with basic auth
	authClient := webdav.HTTPClientWithBasicAuth(httpClient, cfg.Username, cfg.Password)

	// Create WebDAV client
	wdClient, err := webdav.NewClient(authClient, cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	// Create CalDAV client
	cdClient, err := caldav.NewClient(authClient, cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		http:   httpClient,
		webdav: wdClient,
		caldav: cdClient,
		server: cfg,
	}, nil
}

// FindCurrentUserPrincipal discovers the current user's principal URL.
func (c *Client) FindCurrentUserPrincipal(ctx context.Context) (string, error) {
	endpoint, err := normalizeCollectionURL(c.server.ServerURL)
	if err != nil {
		return "", err
	}

	body, err := xml.Marshal(currentUserPrincipalPropFind{
		Prop: currentUserPrincipalPropField{},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", endpoint, strings.NewReader(xml.Header+string(body)))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.server.Username, c.server.Password)
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("Depth", "0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		return "", &multiStatusError{Status: resp.Status}
	}

	var ms currentUserPrincipalMultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return "", err
	}

	for _, response := range ms.Responses {
		for _, propstat := range response.PropStats {
			prop := propstat.Prop.CurrentUserPrincipal
			if prop.Unauthenticated != nil {
				return "", fmt.Errorf("webdav: unauthenticated")
			}
			if prop.Href == "" {
				continue
			}

			href, err := url.Parse(prop.Href)
			if err != nil {
				return "", err
			}
			if href.Path == "" {
				return "", fmt.Errorf("webdav: current-user-principal href is empty")
			}
			return href.Path, nil
		}
	}

	return "", fmt.Errorf("webdav: current-user-principal not found")
}

func normalizeCollectionURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String(), nil
}

// FindCalendarHomeSet discovers the calendar home set from a principal URL.
func (c *Client) FindCalendarHomeSet(ctx context.Context, principal string) (string, error) {
	return c.caldav.FindCalendarHomeSet(ctx, principal)
}

// FindCalendars discovers all calendars in a calendar home set.
func (c *Client) FindCalendars(ctx context.Context, homeSet string) ([]caldav.Calendar, error) {
	return c.caldav.FindCalendars(ctx, homeSet)
}

// HTTPClient returns the underlying HTTP client for direct operations.
func (c *Client) HTTPClient() *http.Client {
	return c.http
}

// CalDAVClient returns the underlying CalDAV client.
func (c *Client) CalDAVClient() *caldav.Client {
	return c.caldav
}

// WebDAVClient returns the underlying WebDAV client.
func (c *Client) WebDAVClient() *webdav.Client {
	return c.webdav
}

// ServerURL returns the configured server URL.
func (c *Client) ServerURL() string {
	return c.server.ServerURL
}
