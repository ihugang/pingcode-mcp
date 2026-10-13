package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests trace to bootstrap-pingcode-mcp task 2.2, PRD FR-002/FR-010/FR-011,
// and the auth-token-provider and secret-redaction specs. PingCode demand ID: PENDING.

const (
	testClientID     = "client-id/with+symbols"
	testClientSecret = "secret-value /+?&-unique"
	testAccessToken  = "access-token-value-unique"
)

func TestRequestContractAndCacheReuse(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/token" {
			t.Errorf("path = %q, want /v1/auth/token", r.URL.Path)
		}
		query := r.URL.Query()
		if got := query.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q", got)
		}
		if got := query.Get("client_id"); got != testClientID {
			t.Errorf("client_id = %q", got)
		}
		if got := query.Get("client_secret"); got != testClientSecret {
			t.Errorf("client_secret query value did not round-trip")
		}
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 {
				t.Errorf("GET request unexpectedly had body %q", body)
			}
		}
		writeTokenResponse(w, testAccessToken, fmt.Sprint(now.Unix()+3600))
	}))
	defer server.Close()

	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL},
		http.DefaultClient,
		WithNow(func() time.Time { return now }),
	)
	for call := 0; call < 2; call++ {
		token, err := provider.AccessToken(context.Background())
		if err != nil {
			t.Fatalf("AccessToken() error = %v", err)
		}
		if token != testAccessToken {
			t.Fatalf("AccessToken() = %q, want %q", token, testAccessToken)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestExpiresInFormats(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	for _, test := range []struct {
		name string
		raw  string
	}{
		{name: "JSON number", raw: fmt.Sprint(now.Unix() + 3600)},
		{name: "decimal string", raw: fmt.Sprintf("%q", fmt.Sprint(now.Unix()+3600))},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeTokenResponse(w, testAccessToken, test.raw)
			}))
			defer server.Close()
			provider := NewClientCredentialsProvider(
				Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL},
				nil,
				WithNow(func() time.Time { return now }),
			)
			token, err := provider.AccessToken(context.Background())
			if err != nil || token != testAccessToken {
				t.Fatalf("AccessToken() = (%q, %v), want token", token, err)
			}
		})
	}
}

func TestTokenTypeIsOptionalAndNotValidated(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "omitted",
			body: fmt.Sprintf(`{"access_token":%q,"expires_in":%d}`, testAccessToken, now.Unix()+3600),
		},
		{
			name: "non-string value",
			body: fmt.Sprintf(`{"access_token":%q,"token_type":{"ignored":true},"expires_in":%d}`, testAccessToken, now.Unix()+3600),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			provider := NewClientCredentialsProvider(
				Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
				WithNow(func() time.Time { return now }),
			)
			token, err := provider.AccessToken(context.Background())
			if err != nil || token != testAccessToken {
				t.Fatalf("AccessToken() = (%q, %v), want token", token, err)
			}
		})
	}
}

func TestInvalidExpiresInFailsClosedAndCanRetry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "non-decimal", body: `{"access_token":"` + testAccessToken + `","expires_in":"abc"}`},
		{name: "missing", body: `{"access_token":"` + testAccessToken + `"}`},
		{name: "past absolute time", body: `{"access_token":"` + testAccessToken + `","expires_in":1}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if requests.Add(1) == 1 {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, test.body)
					return
				}
				writeTokenResponse(w, "retry-token", fmt.Sprint(now.Unix()+3600))
			}))
			defer server.Close()
			provider := NewClientCredentialsProvider(
				Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL},
				nil,
				WithNow(func() time.Time { return now }),
			)

			_, err := provider.AccessToken(context.Background())
			var authErr *Error
			if !errors.As(err, &authErr) || authErr.Kind != KindInvalidResponse {
				t.Fatalf("first AccessToken() error = %v, want INVALID_RESPONSE", err)
			}
			assertErrorRedacted(t, err, testClientID, testClientSecret, testAccessToken, "retry-token")
			token, err := provider.AccessToken(context.Background())
			if err != nil || token != "retry-token" {
				t.Fatalf("retry AccessToken() = (%q, %v), want retry token", token, err)
			}
			if got := requests.Load(); got != 2 {
				t.Fatalf("request count = %d, want 2", got)
			}
		})
	}
}

func TestAuthenticationErrorIsClassifiedAndRedacted(t *testing.T) {
	const responseToken = "response-token-from-error-unique"
	requestURL := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL <- r.URL.String()
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"100022","message":"`+testClientSecret+` `+responseToken+`"}`)
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
	)

	_, err := provider.AccessToken(context.Background())
	var authErr *Error
	if !errors.As(err, &authErr) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if authErr.Kind != KindAuthentication || authErr.Status != http.StatusBadRequest || authErr.APICode != "100022" {
		t.Fatalf("error metadata = (%q, %d, %q)", authErr.Kind, authErr.Status, authErr.APICode)
	}
	assertErrorRedacted(t, err, testClientID, testClientSecret, url.QueryEscape(testClientSecret), responseToken, <-requestURL)
}

func TestTransportFailureIsClassifiedAndRedacted(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	baseURL := server.URL
	server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: baseURL}, nil,
	)

	_, err := provider.AccessToken(context.Background())
	var authErr *Error
	if !errors.As(err, &authErr) || authErr.Kind != KindUpstream {
		t.Fatalf("error = %v, want UPSTREAM", err)
	}
	assertErrorRedacted(t, err, testClientID, testClientSecret, url.QueryEscape(testClientSecret), baseURL)
}

func TestMissingCredentialsDoNotMakeRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: "", ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
	)

	_, err := provider.AccessToken(context.Background())
	var authErr *Error
	if !errors.As(err, &authErr) || authErr.Kind != KindNotConfigured {
		t.Fatalf("error = %v, want NOT_CONFIGURED", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
	assertErrorRedacted(t, err, testClientSecret, url.QueryEscape(testClientSecret))
}

func TestRefreshWindowUsesInjectedClock(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	clock := &testClock{now: t0}
	expires := t0.Add(time.Hour).Unix()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := requests.Add(1)
		if count >= 3 {
			writeTokenResponse(w, fmt.Sprintf("token-%d", count), fmt.Sprint(expires+3600))
			return
		}
		writeTokenResponse(w, fmt.Sprintf("token-%d", count), fmt.Sprint(expires))
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
		WithNow(clock.Now),
	)

	first, err := provider.AccessToken(context.Background())
	if err != nil || first != "token-1" {
		t.Fatalf("first token = (%q, %v)", first, err)
	}
	clock.Set(t0.Add(time.Hour - 59*time.Second))
	second, err := provider.AccessToken(context.Background())
	if err != nil || second != "token-2" {
		t.Fatalf("refresh-window token = (%q, %v)", second, err)
	}
	clock.Set(t0.Add(time.Hour))
	third, err := provider.AccessToken(context.Background())
	if err != nil || third != "token-3" {
		t.Fatalf("expired token refresh = (%q, %v)", third, err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
}

func TestConcurrentInitialCallsShareOneRefresh(t *testing.T) {
	const callers = 20
	now := time.Unix(1_800_000_000, 0)
	entered := make(chan struct{})
	release := make(chan struct{})
	var requests atomic.Int32
	var enteredOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		enteredOnce.Do(func() { close(entered) })
		<-release
		writeTokenResponse(w, testAccessToken, fmt.Sprint(now.Unix()+3600))
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
		WithNow(func() time.Time { return now }),
	)

	start := make(chan struct{})
	started := make(chan struct{}, callers)
	type result struct {
		token string
		err   error
	}
	results := make(chan result, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			started <- struct{}{}
			token, err := provider.AccessToken(context.Background())
			results <- result{token: token, err: err}
		}()
	}
	close(start)
	for i := 0; i < callers; i++ {
		<-started
	}
	<-entered
	close(release)

	for i := 0; i < callers; i++ {
		result := <-results
		if result.err != nil || result.token != testAccessToken {
			t.Errorf("caller result = (%q, %v), want shared token", result.token, result.err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestRefreshFailureIsSharedAndCanRetry(t *testing.T) {
	const waiters = 5
	now := time.Unix(1_800_000_000, 0)
	entered := make(chan struct{})
	release := make(chan struct{})
	var requests atomic.Int32
	var enteredOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			enteredOnce.Do(func() { close(entered) })
			<-release
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"code":"100022","message":"`+testClientSecret+`"}`)
			return
		}
		writeTokenResponse(w, "retry-after-shared-error", fmt.Sprint(now.Unix()+3600))
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
		WithNow(func() time.Time { return now }),
	)

	type result struct {
		token string
		err   error
	}
	ownerResult := make(chan result, 1)
	go func() {
		token, err := provider.AccessToken(context.Background())
		ownerResult <- result{token: token, err: err}
	}()
	<-entered

	waitContexts := make([]*observedContext, waiters)
	waiterResults := make(chan result, waiters)
	for i := range waitContexts {
		base := context.Background()
		waitContexts[i] = &observedContext{Context: base, doneRead: make(chan struct{})}
		go func(ctx context.Context) {
			token, err := provider.AccessToken(ctx)
			waiterResults <- result{token: token, err: err}
		}(waitContexts[i])
	}
	for _, ctx := range waitContexts {
		<-ctx.doneRead
	}
	close(release)

	owner := <-ownerResult
	if owner.err == nil {
		t.Fatal("refresh owner unexpectedly succeeded")
	}
	var sharedErr *Error
	if !errors.As(owner.err, &sharedErr) || sharedErr.Kind != KindAuthentication {
		t.Fatalf("owner error = %v, want AUTHENTICATION", owner.err)
	}
	for i := 0; i < waiters; i++ {
		waiter := <-waiterResults
		if waiter.err != owner.err {
			t.Errorf("waiter %d did not receive the shared error instance", i)
		}
	}
	assertErrorRedacted(t, owner.err, testClientID, testClientSecret, url.QueryEscape(testClientSecret))

	token, err := provider.AccessToken(context.Background())
	if err != nil || token != "retry-after-shared-error" {
		t.Fatalf("retry AccessToken() = (%q, %v)", token, err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("request count = %d, want failed refresh plus retry", got)
	}
}

func TestWaitingCallerCanCancelWithoutCancelingRefresh(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enteredOnce.Do(func() { close(entered) })
		select {
		case <-release:
			writeTokenResponse(w, testAccessToken, fmt.Sprint(now.Unix()+3600))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
		WithNow(func() time.Time { return now }),
	)

	type result struct {
		token string
		err   error
	}
	ownerResult := make(chan result, 1)
	go func() {
		token, err := provider.AccessToken(context.Background())
		ownerResult <- result{token: token, err: err}
	}()
	<-entered

	baseCtx, cancel := context.WithCancel(context.Background())
	waitingCtx := &observedContext{Context: baseCtx, doneRead: make(chan struct{})}
	waiterResult := make(chan result, 1)
	go func() {
		token, err := provider.AccessToken(waitingCtx)
		waiterResult <- result{token: token, err: err}
	}()
	<-waitingCtx.doneRead // the caller has joined the flight's cancellable wait
	cancel()

	select {
	case result := <-waiterResult:
		if !errors.Is(result.err, context.Canceled) {
			t.Fatalf("waiter error = %v, want context.Canceled", result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled waiter did not return promptly")
	}

	close(release)
	select {
	case result := <-ownerResult:
		if result.err != nil || result.token != testAccessToken {
			t.Fatalf("refresh owner result = (%q, %v), want token", result.token, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("refresh owner did not complete after release")
	}
}

func TestRequestContextCancellationStopsHTTPRequest(t *testing.T) {
	entered := make(chan struct{})
	requestCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(requestCanceled)
	}))
	defer server.Close()
	provider := NewClientCredentialsProvider(
		Credentials{ClientID: testClientID, ClientSecret: testClientSecret, BaseURL: server.URL}, nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := provider.AccessToken(ctx)
		result <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("AccessToken() error = %v, want context.Canceled", err)
		}
		assertErrorRedacted(t, err, testClientID, testClientSecret, url.QueryEscape(testClientSecret))
	case <-time.After(time.Second):
		t.Fatal("canceled request did not return promptly")
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("HTTP handler did not observe request context cancellation")
	}
}

func writeTokenResponse(w http.ResponseWriter, token, expiresRaw string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":%s}`, token, expiresRaw)
}

func assertErrorRedacted(t *testing.T, err error, forbidden ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for current := err; current != nil; current = errors.Unwrap(current) {
		message := current.Error()
		for _, value := range forbidden {
			if value != "" && strings.Contains(message, value) {
				t.Errorf("error chain contains sensitive value %q: %q", value, message)
			}
		}
	}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *testClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *testClock) Set(now time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = now
}

type observedContext struct {
	context.Context
	doneRead chan struct{}
	once     sync.Once
}

func (ctx *observedContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.doneRead) })
	return ctx.Context.Done()
}
