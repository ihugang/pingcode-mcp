package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://open.pingcode.com"
	defaultHTTPTimeout = 30 * time.Second
	defaultLogLevel    = "info"
)

// Config contains process settings for OpenSpec bootstrap-pingcode-mcp task 2.1,
// supporting PRD FR-002 and FR-010. Credentials may be empty when the application
// is assembled for offline use.
type Config struct {
	ClientID          string
	ClientSecret      string
	BaseURL           string
	ReadOnly          bool
	AllowedProjectIDs []string
	HTTPTimeout       time.Duration
	LogLevel          string
}

// Load reads and validates configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.LookupEnv)
}

// LoadFrom reads and validates configuration using lookup.
// It is intended to make configuration loading testable without changing the process environment.
func LoadFrom(lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("config: environment lookup function is nil")
	}

	config := Config{
		ClientID:          lookupValue(lookup, "PINGCODE_CLIENT_ID"),
		ClientSecret:      lookupValue(lookup, "PINGCODE_CLIENT_SECRET"),
		BaseURL:           defaultBaseURL,
		ReadOnly:          true,
		AllowedProjectIDs: []string{},
		HTTPTimeout:       defaultHTTPTimeout,
		LogLevel:          defaultLogLevel,
	}

	if (config.ClientID == "") != (config.ClientSecret == "") {
		return Config{}, configError(config.ClientSecret, "PINGCODE_CLIENT_ID and PINGCODE_CLIENT_SECRET must either both be set or both be empty")
	}

	if value, ok := lookup("PINGCODE_BASE_URL"); ok && value != "" {
		config.BaseURL = strings.TrimRight(value, "/")
	}
	parsedBaseURL, err := url.Parse(config.BaseURL)
	if err != nil || !parsedBaseURL.IsAbs() || parsedBaseURL.Host == "" ||
		(!strings.EqualFold(parsedBaseURL.Scheme, "http") && !strings.EqualFold(parsedBaseURL.Scheme, "https")) ||
		parsedBaseURL.RawQuery != "" || parsedBaseURL.ForceQuery || parsedBaseURL.Fragment != "" ||
		strings.Contains(config.BaseURL, "#") {
		return Config{}, configError(config.ClientSecret, "PINGCODE_BASE_URL must be an absolute http or https URL with a host and without query or fragment")
	}

	if value, ok := lookup("PINGCODE_READ_ONLY"); ok && value != "" {
		readOnly, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return Config{}, configError(config.ClientSecret, "PINGCODE_READ_ONLY has invalid boolean value")
		}
		config.ReadOnly = readOnly
	}

	if value, ok := lookup("PINGCODE_ALLOWED_PROJECT_IDS"); ok && value != "" {
		projectIDs := strings.Split(value, ",")
		config.AllowedProjectIDs = make([]string, 0, len(projectIDs))
		for _, projectID := range projectIDs {
			projectID = strings.TrimSpace(projectID)
			if projectID == "" {
				return Config{}, configError(config.ClientSecret, "PINGCODE_ALLOWED_PROJECT_IDS must not contain empty project IDs")
			}
			config.AllowedProjectIDs = append(config.AllowedProjectIDs, projectID)
		}
	}

	if value, ok := lookup("PINGCODE_HTTP_TIMEOUT"); ok && value != "" {
		timeout, parseErr := time.ParseDuration(value)
		if parseErr != nil || timeout <= 0 {
			return Config{}, configError(config.ClientSecret, "PINGCODE_HTTP_TIMEOUT must be a positive duration")
		}
		config.HTTPTimeout = timeout
	}

	if value, ok := lookup("PINGCODE_LOG_LEVEL"); ok && value != "" {
		logLevel := strings.ToLower(value)
		switch logLevel {
		case "debug", "info", "warn", "error":
			config.LogLevel = logLevel
		default:
			return Config{}, configError(config.ClientSecret, "PINGCODE_LOG_LEVEL has unsupported value")
		}
	}

	return config, nil
}

func lookupValue(lookup func(string) (string, bool), key string) string {
	value, _ := lookup(key)
	return value
}

func configError(clientSecret, message string) error {
	if clientSecret != "" {
		message = strings.ReplaceAll(message, clientSecret, "")
	}
	return errors.New(message)
}
