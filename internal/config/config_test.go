package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests trace OpenSpec bootstrap-pingcode-mcp task 2.1 to PRD FR-002 and FR-010.

func TestLoadFromDefaults(t *testing.T) {
	config, err := LoadFrom(mapLookup(nil))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	want := Config{
		BaseURL:           defaultBaseURL,
		ReadOnly:          true,
		AllowedProjectIDs: []string{},
		HTTPTimeout:       30 * time.Second,
		LogLevel:          "info",
	}
	if !reflect.DeepEqual(config, want) {
		t.Fatalf("LoadFrom() = %#v, want %#v", config, want)
	}
}

func TestLoadFromEmptyValuesUseDefaults(t *testing.T) {
	config, err := LoadFrom(mapLookup(map[string]string{
		"PINGCODE_BASE_URL":            "",
		"PINGCODE_READ_ONLY":           "",
		"PINGCODE_ALLOWED_PROJECT_IDS": "",
		"PINGCODE_HTTP_TIMEOUT":        "",
		"PINGCODE_LOG_LEVEL":           "",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if config.BaseURL != defaultBaseURL || !config.ReadOnly || len(config.AllowedProjectIDs) != 0 ||
		config.HTTPTimeout != 30*time.Second || config.LogLevel != "info" {
		t.Fatalf("LoadFrom() with empty values = %#v, want defaults", config)
	}
}

func TestLoadFromOverrides(t *testing.T) {
	config, err := LoadFrom(mapLookup(map[string]string{
		"PINGCODE_CLIENT_ID":           "client-id",
		"PINGCODE_CLIENT_SECRET":       "client-secret",
		"PINGCODE_BASE_URL":            "https://pingcode.example/open////",
		"PINGCODE_READ_ONLY":           "false",
		"PINGCODE_ALLOWED_PROJECT_IDS": " alpha , beta ",
		"PINGCODE_HTTP_TIMEOUT":        "5s",
		"PINGCODE_LOG_LEVEL":           "Debug",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	want := Config{
		ClientID:          "client-id",
		ClientSecret:      "client-secret",
		BaseURL:           "https://pingcode.example/open",
		ReadOnly:          false,
		AllowedProjectIDs: []string{"alpha", "beta"},
		HTTPTimeout:       5 * time.Second,
		LogLevel:          "debug",
	}
	if !reflect.DeepEqual(config, want) {
		t.Fatalf("LoadFrom() = %#v, want %#v", config, want)
	}
}

func TestLoadFromReadOnlyValues(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "true", want: true},
		{value: "false", want: false},
		{value: "TRUE", want: true},
		{value: "0", want: false},
		{value: "1", want: true},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			config, err := LoadFrom(mapLookup(map[string]string{"PINGCODE_READ_ONLY": test.value}))
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if config.ReadOnly != test.want {
				t.Fatalf("ReadOnly = %t, want %t", config.ReadOnly, test.want)
			}
		})
	}

	config, err := LoadFrom(mapLookup(map[string]string{"PINGCODE_READ_ONLY": ""}))
	if err != nil {
		t.Fatalf("LoadFrom() with empty value error = %v", err)
	}
	if !config.ReadOnly {
		t.Fatal("ReadOnly with empty value = false, want true")
	}
}

func TestLoadFromLogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			config, err := LoadFrom(mapLookup(map[string]string{"PINGCODE_LOG_LEVEL": strings.ToUpper(level)}))
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if config.LogLevel != level {
				t.Fatalf("LogLevel = %q, want %q", config.LogLevel, level)
			}
		})
	}
}

func TestLoadFromAllowedProjectIDs(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []string
		wantErr bool
	}{
		{name: "trims items and preserves order", value: " a , b ", want: []string{"a", "b"}},
		{name: "empty string", value: "", want: []string{}},
		{name: "rejects empty item", value: "a,,b", wantErr: true},
		{name: "rejects whitespace item", value: "a,  ,b", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := LoadFrom(mapLookup(map[string]string{"PINGCODE_ALLOWED_PROJECT_IDS": test.value}))
			if test.wantErr {
				if err == nil {
					t.Fatal("LoadFrom() error = nil, want an error")
				}
				if !strings.Contains(err.Error(), "PINGCODE_ALLOWED_PROJECT_IDS") {
					t.Fatalf("error %q does not name PINGCODE_ALLOWED_PROJECT_IDS", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if !reflect.DeepEqual(config.AllowedProjectIDs, test.want) {
				t.Fatalf("AllowedProjectIDs = %#v, want %#v", config.AllowedProjectIDs, test.want)
			}
		})
	}
}

func TestLoadFromCredentials(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		wantErr bool
	}{
		{name: "both missing is allowed", values: map[string]string{}},
		{name: "both empty is allowed", values: map[string]string{"PINGCODE_CLIENT_ID": "", "PINGCODE_CLIENT_SECRET": ""}},
		{name: "client ID only is rejected", values: map[string]string{"PINGCODE_CLIENT_ID": "client-id"}, wantErr: true},
		{name: "client secret only is rejected", values: map[string]string{"PINGCODE_CLIENT_SECRET": "s3cr3t-config-test-unique"}, wantErr: true},
		{name: "both set is allowed", values: map[string]string{"PINGCODE_CLIENT_ID": "client-id", "PINGCODE_CLIENT_SECRET": "client-secret"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadFrom(mapLookup(test.values))
			if (err != nil) != test.wantErr {
				t.Fatalf("LoadFrom() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestLoadFromRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "read only boolean", key: "PINGCODE_READ_ONLY", value: "yes"},
		{name: "timeout syntax", key: "PINGCODE_HTTP_TIMEOUT", value: "abc"},
		{name: "zero timeout", key: "PINGCODE_HTTP_TIMEOUT", value: "0s"},
		{name: "negative timeout", key: "PINGCODE_HTTP_TIMEOUT", value: "-1s"},
		{name: "unsupported log level", key: "PINGCODE_LOG_LEVEL", value: "verbose"},
		{name: "FTP URL", key: "PINGCODE_BASE_URL", value: "ftp://pingcode.example"},
		{name: "relative URL", key: "PINGCODE_BASE_URL", value: "/open"},
		{name: "URL with query", key: "PINGCODE_BASE_URL", value: "https://pingcode.example/open?tenant=x"},
		{name: "URL with empty query", key: "PINGCODE_BASE_URL", value: "https://pingcode.example/open?"},
		{name: "URL with fragment", key: "PINGCODE_BASE_URL", value: "https://pingcode.example/open#section"},
		{name: "URL with empty fragment", key: "PINGCODE_BASE_URL", value: "https://pingcode.example/open#"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadFrom(mapLookup(map[string]string{test.key: test.value}))
			if err == nil {
				t.Fatal("LoadFrom() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.key) {
				t.Fatalf("error %q does not name %s", err, test.key)
			}
		})
	}
}

func TestLoadFromErrorsNeverContainClientSecret(t *testing.T) {
	const secret = "unique-secret-must-never-appear-in-config-errors-9f50d31"
	tests := []struct {
		name   string
		values map[string]string
	}{
		{
			name: "one credential missing",
			values: map[string]string{
				"PINGCODE_CLIENT_SECRET": secret,
			},
		},
		{
			name: "base URL validation error",
			values: map[string]string{
				"PINGCODE_CLIENT_ID":     "client-id",
				"PINGCODE_CLIENT_SECRET": secret,
				"PINGCODE_BASE_URL":      "ftp://invalid.example",
			},
		},
		{
			name: "boolean validation error",
			values: map[string]string{
				"PINGCODE_CLIENT_ID":     "client-id",
				"PINGCODE_CLIENT_SECRET": secret,
				"PINGCODE_READ_ONLY":     "yes",
			},
		},
		{
			name: "secret matches a non-secret invalid value",
			values: map[string]string{
				"PINGCODE_CLIENT_ID":     "client-id",
				"PINGCODE_CLIENT_SECRET": "yes",
				"PINGCODE_READ_ONLY":     "yes",
			},
		},
		{
			name: "timeout validation error",
			values: map[string]string{
				"PINGCODE_CLIENT_ID":     "client-id",
				"PINGCODE_CLIENT_SECRET": secret,
				"PINGCODE_HTTP_TIMEOUT":  "0s",
			},
		},
		{
			name: "log level validation error",
			values: map[string]string{
				"PINGCODE_CLIENT_ID":     "client-id",
				"PINGCODE_CLIENT_SECRET": secret,
				"PINGCODE_LOG_LEVEL":     "verbose",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadFrom(mapLookup(test.values))
			if err == nil {
				t.Fatal("LoadFrom() error = nil, want an error")
			}
			secret := test.values["PINGCODE_CLIENT_SECRET"]
			for cause := err; cause != nil; cause = errors.Unwrap(cause) {
				if secret != "" && strings.Contains(cause.Error(), secret) {
					t.Fatalf("error chain contains client secret")
				}
			}
		})
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
