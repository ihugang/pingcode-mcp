package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// This implementation is tracked by bootstrap-pingcode-mcp task 2.2 and PRD
// FR-002/FR-010. Its contract is defined by auth-token-provider and
// secret-redaction; PingCode demand ID: PENDING.

const (
	refreshWindow       = 60 * time.Second
	maxErrorBodyBytes   = 64 << 10
	maxTokenBodyBytes   = 1 << 20
	tokenEndpointSuffix = "/v1/auth/token"
)

var errUpstream = errors.New("auth: upstream request failed")

// TokenProvider 是上层获取访问令牌的唯一抽象。
type TokenProvider interface {
	AccessToken(ctx context.Context) (string, error)
}

// Credentials 携带 Client Credentials 配置，值通常来自 internal/config。
type Credentials struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
}

// Option configures a ClientCredentialsProvider.
type Option func(*ClientCredentialsProvider)

// WithNow injects the clock used for expiry checks. A nil function is ignored.
func WithNow(now func() time.Time) Option {
	return func(provider *ClientCredentialsProvider) {
		if now != nil {
			provider.now = now
		}
	}
}

// ErrorKind classifies errors returned by the token provider.
type ErrorKind string

const (
	KindNotConfigured   ErrorKind = "NOT_CONFIGURED"
	KindAuthentication  ErrorKind = "AUTHENTICATION"
	KindUpstream        ErrorKind = "UPSTREAM"
	KindInvalidResponse ErrorKind = "INVALID_RESPONSE"
)

// Error is a safe, classifiable token-provider error. Its cause is kept private
// so transport errors containing request URLs or query values are never exposed.
type Error struct {
	Kind    ErrorKind
	Status  int
	APICode string

	cause      error
	redactions []string
}

// Error returns a stable message containing only the error class and safe
// status metadata.
func (e *Error) Error() string {
	if e == nil {
		return "auth: token provider failed"
	}

	message := "auth: token provider failed"
	switch e.Kind {
	case KindNotConfigured:
		message = "auth: NOT_CONFIGURED: client credentials are not configured"
	case KindAuthentication:
		message = "auth: AUTHENTICATION: token endpoint rejected the request"
	case KindUpstream:
		message = "auth: UPSTREAM: token endpoint request failed"
	case KindInvalidResponse:
		message = "auth: INVALID_RESPONSE: token endpoint returned an invalid response"
	}

	if e.Status > 0 {
		message += fmt.Sprintf(" (HTTP %d", e.Status)
		if code := safeCodeText(e.APICode, e.redactions); code != "" {
			message += ", code " + code
		}
		message += ")"
	}
	return redact(message, e.redactions)
}

// Unwrap returns a sanitized cause suitable for errors.Is checks.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

type tokenCache struct {
	accessToken string
	expiresAt   time.Time
}

type refreshFlight struct {
	done  chan struct{}
	token string
	err   error
}

// ClientCredentialsProvider fetches, caches, and refreshes PingCode enterprise
// access tokens using the Client Credentials grant.
type ClientCredentialsProvider struct {
	creds Credentials
	http  *http.Client
	now   func() time.Time

	mu     sync.Mutex
	cache  tokenCache
	flight *refreshFlight
}

// NewClientCredentialsProvider creates a Client Credentials token provider.
// A nil HTTP client uses http.DefaultClient.
func NewClientCredentialsProvider(creds Credentials, httpClient *http.Client, opts ...Option) *ClientCredentialsProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	provider := &ClientCredentialsProvider{
		creds: creds,
		http:  httpClient,
		now:   time.Now,
	}
	for _, option := range opts {
		if option != nil {
			option(provider)
		}
	}
	return provider
}

// AccessToken returns a cached token outside the 60-second refresh window, or
// joins/starts one cancellable token refresh.
func (p *ClientCredentialsProvider) AccessToken(ctx context.Context) (string, error) {
	if p.creds.ClientID == "" || p.creds.ClientSecret == "" {
		return "", &Error{Kind: KindNotConfigured}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	p.mu.Lock()
	now := p.now()
	if p.cache.accessToken != "" && now.Before(p.cache.expiresAt.Add(-refreshWindow)) {
		token := p.cache.accessToken
		p.mu.Unlock()
		return token, nil
	}

	if active := p.flight; active != nil {
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-active.done:
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return active.token, active.err
		}
	}

	flight := &refreshFlight{done: make(chan struct{})}
	p.flight = flight
	knownToken := p.cache.accessToken
	p.mu.Unlock()

	token, expiresAt, err := p.fetch(ctx, knownToken)

	p.mu.Lock()
	if err == nil {
		p.cache = tokenCache{accessToken: token, expiresAt: expiresAt}
	}
	flight.token = token
	flight.err = err
	p.flight = nil
	close(flight.done)
	p.mu.Unlock()

	return token, err
}

func (p *ClientCredentialsProvider) fetch(ctx context.Context, knownToken string) (string, time.Time, error) {
	endpoint, err := url.Parse(strings.TrimRight(p.creds.BaseURL, "/") + tokenEndpointSuffix)
	if err != nil || !endpoint.IsAbs() || endpoint.Host == "" ||
		(!strings.EqualFold(endpoint.Scheme, "http") && !strings.EqualFold(endpoint.Scheme, "https")) ||
		endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return "", time.Time{}, newError(KindUpstream, 0, "", errUpstream, p.redactions(knownToken))
	}

	query := url.Values{}
	query.Set("grant_type", "client_credentials")
	query.Set("client_id", p.creds.ClientID)
	query.Set("client_secret", p.creds.ClientSecret)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", time.Time{}, newError(KindUpstream, 0, "", errUpstream, p.redactions(knownToken))
	}

	// OAuth credentials are in the query by contract. Do not forward them to a
	// redirect target, even if the caller's shared client follows redirects.
	client := *p.http
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := client.Do(request)
	if err != nil {
		cause := sanitizedTransportCause(ctx, err)
		return "", time.Time{}, newError(KindUpstream, 0, "", cause, p.redactions(knownToken))
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		code, responseToken := errorEnvelopeMetadata(body)
		redactions := p.redactions(knownToken, responseToken)
		code = safeCodeText(code, redactions)
		return "", time.Time{}, newError(KindAuthentication, response.StatusCode, code, nil, redactions)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTokenBodyBytes+1))
	if err != nil {
		return "", time.Time{}, newError(KindUpstream, response.StatusCode, "", sanitizedTransportCause(ctx, err), p.redactions(knownToken))
	}
	if len(body) > maxTokenBodyBytes {
		return "", time.Time{}, newError(KindInvalidResponse, response.StatusCode, "", nil, p.redactions(knownToken))
	}

	var payload struct {
		AccessToken string          `json:"access_token"`
		TokenType   json.RawMessage `json:"token_type"`
		ExpiresIn   json.RawMessage `json:"expires_in"`
	}
	if json.Unmarshal(body, &payload) != nil || strings.TrimSpace(payload.AccessToken) == "" {
		return "", time.Time{}, newError(KindInvalidResponse, response.StatusCode, "", nil, p.redactions(knownToken))
	}

	expiresSeconds, ok := parseAbsoluteUnixSeconds(payload.ExpiresIn)
	if !ok {
		return "", time.Time{}, newError(KindInvalidResponse, response.StatusCode, "", nil, p.redactions(knownToken, payload.AccessToken))
	}
	expiresAt := time.Unix(expiresSeconds, 0)
	if !expiresAt.After(p.now()) {
		return "", time.Time{}, newError(KindInvalidResponse, response.StatusCode, "", nil, p.redactions(knownToken, payload.AccessToken))
	}
	return payload.AccessToken, expiresAt, nil
}

func parseAbsoluteUnixSeconds(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}

	value := string(raw)
	if value[0] == '"' {
		if json.Unmarshal(raw, &value) != nil {
			return 0, false
		}
	}
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	return seconds, err == nil
}

func errorEnvelopeMetadata(body []byte) (code, responseToken string) {
	var envelope struct {
		Code        json.RawMessage `json:"code"`
		AccessToken string          `json:"access_token"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return "", ""
	}
	return rawCode(envelope.Code), envelope.AccessToken
}

func rawCode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var code string
		if json.Unmarshal(raw, &code) == nil {
			return code
		}
		return ""
	}
	return string(raw)
}

func safeCodeText(code string, redactions []string) string {
	if len(code) == 0 || len(code) > 64 {
		return ""
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '_' && r != '-' && r != '.' {
			return ""
		}
	}
	lowerCode := strings.ToLower(code)
	if strings.Contains(lowerCode, "authorization") || strings.Contains(lowerCode, "bearer") {
		return ""
	}
	for _, value := range redactions {
		if value != "" && strings.Contains(code, value) {
			return ""
		}
	}
	return code
}

func (p *ClientCredentialsProvider) redactions(additional ...string) []string {
	values := []string{p.creds.ClientID, p.creds.ClientSecret}
	values = append(values, additional...)
	redactions := make([]string, 0, len(values)*3)
	for _, value := range values {
		if value == "" {
			continue
		}
		redactions = append(redactions, value, url.QueryEscape(value), url.PathEscape(value))
	}
	return redactions
}

func newError(kind ErrorKind, status int, code string, cause error, redactions []string) *Error {
	return &Error{Kind: kind, Status: status, APICode: code, cause: cause, redactions: redactions}
}

func sanitizedTransportCause(ctx context.Context, cause error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(cause, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errUpstream
}

func redact(message string, values []string) string {
	for _, value := range values {
		if value != "" {
			message = strings.ReplaceAll(message, value, "[REDACTED]")
		}
	}
	return message
}
