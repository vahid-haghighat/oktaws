package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AuthFlow            string `yaml:"auth_flow"`
	OrgDomain           string `yaml:"org_domain"`
	OIDCClientID        string `yaml:"oidc_client_id"`
	AWSIAMRole          string `yaml:"aws_iam_role"`
	AWSIAMIdP           string `yaml:"aws_iam_idp"`
	AWSAcctFedAppID     string `yaml:"aws_acct_fed_app_id"`
	Profile             string `yaml:"profile"`
	SessionDuration     int    `yaml:"session_duration"`
	Format              string `yaml:"format"`
	AWSRegion           string `yaml:"aws_region"`
	QRCode              bool   `yaml:"qr_code"`
	OpenBrowser         bool   `yaml:"open_browser"`
	OpenBrowserCommand  string `yaml:"open_browser_command"`
	AllProfiles         bool   `yaml:"all_profiles"`
	WriteAWSCredentials bool   `yaml:"write_aws_credentials"`
	CacheAccessToken    bool   `yaml:"cache_access_token"`
	Debug               bool   `yaml:"debug"`
	DebugAPICalls       bool   `yaml:"debug_api_calls"`
}

func NewConfig() (*Config, error) {
	cfg, err := LoadConfigFromFile()
	if err != nil {
		cfg = &Config{}
	}
	cfg.MergeWithViper()
	if cfg.SessionDuration == 0 {
		cfg.SessionDuration = 3600
	}
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	if cfg.Format == "" {
		cfg.Format = "env-var"
	}
	if cfg.AuthFlow == "" {
		cfg.AuthFlow = "auto"
	}
	return cfg, nil
}
func (c *Config) MergeWithViper() {
	if v := viper.GetString("auth-flow"); v != "" {
		c.AuthFlow = v
	}
	if v := viper.GetString("org-domain"); v != "" {
		c.OrgDomain = v
	}
	if v := viper.GetString("oidc-client-id"); v != "" {
		c.OIDCClientID = v
	}
	if v := viper.GetString("aws-iam-role"); v != "" {
		c.AWSIAMRole = v
	}
	if v := viper.GetString("aws-iam-idp"); v != "" {
		c.AWSIAMIdP = v
	}
	if v := viper.GetString("aws-acct-fed-app-id"); v != "" {
		c.AWSAcctFedAppID = v
	}
	if v := viper.GetString("profile"); v != "" {
		c.Profile = v
	}
	if v := viper.GetString("format"); v != "" {
		c.Format = v
	}
	if v := viper.GetString("aws-region"); v != "" {
		c.AWSRegion = v
	}
	if v := viper.GetString("open-browser-command"); v != "" {
		c.OpenBrowserCommand = v
	}
	if durationStr := viper.GetString("aws-session-duration"); durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil {
			c.SessionDuration = duration
		}
	}
	if viper.IsSet("qr-code") {
		c.QRCode = viper.GetBool("qr-code")
	}
	if viper.IsSet("open-browser") {
		c.OpenBrowser = viper.GetBool("open-browser")
	}
	if viper.IsSet("all-profiles") {
		c.AllProfiles = viper.GetBool("all-profiles")
	}
	if viper.IsSet("write-aws-credentials") {
		c.WriteAWSCredentials = viper.GetBool("write-aws-credentials")
	}
	if viper.IsSet("cache-access-token") {
		c.CacheAccessToken = viper.GetBool("cache-access-token")
	}
	if viper.IsSet("debug") {
		c.Debug = viper.GetBool("debug")
	}
	if viper.IsSet("debug-api-calls") {
		c.DebugAPICalls = viper.GetBool("debug-api-calls")
	}
}
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "oktaws", "config.yaml")
}
func LoadConfigFromFile() (*Config, error) {
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
func (c *Config) SaveToFile() error {
	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}
func (c *Config) SetValue(key, value string) error {
	key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	switch key {
	case "auth_flow":
		if value != "auto" && value != "oidc" && value != "saml-browser" && value != "saml_browser" {
			return fmt.Errorf("invalid auth_flow: must be 'auto', 'oidc', or 'saml-browser'")
		}
		c.AuthFlow = value
	case "org_domain":
		c.OrgDomain = value
	case "oidc_client_id":
		c.OIDCClientID = value
	case "aws_iam_role":
		c.AWSIAMRole = value
	case "aws_iam_idp":
		c.AWSIAMIdP = value
	case "aws_acct_fed_app_id":
		c.AWSAcctFedAppID = value
	case "profile":
		c.Profile = value
	case "format":
		c.Format = value
	case "aws_region":
		c.AWSRegion = value
	case "open_browser_command":
		c.OpenBrowserCommand = value
	case "session_duration":
		duration, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid session_duration: must be a number")
		}
		c.SessionDuration = duration
	case "qr_code":
		c.QRCode = value == "true" || value == "yes" || value == "1"
	case "open_browser":
		c.OpenBrowser = value == "true" || value == "yes" || value == "1"
	case "all_profiles":
		c.AllProfiles = value == "true" || value == "yes" || value == "1"
	case "write_aws_credentials":
		c.WriteAWSCredentials = value == "true" || value == "yes" || value == "1"
	case "cache_access_token":
		c.CacheAccessToken = value == "true" || value == "yes" || value == "1"
	case "debug":
		c.Debug = value == "true" || value == "yes" || value == "1"
	case "debug_api_calls":
		c.DebugAPICalls = value == "true" || value == "yes" || value == "1"
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}
func (c *Config) GetValue(key string) (string, error) {
	key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	switch key {
	case "auth_flow":
		return c.AuthFlow, nil
	case "org_domain":
		return c.OrgDomain, nil
	case "oidc_client_id":
		return c.OIDCClientID, nil
	case "aws_iam_role":
		return c.AWSIAMRole, nil
	case "aws_iam_idp":
		return c.AWSIAMIdP, nil
	case "aws_acct_fed_app_id":
		return c.AWSAcctFedAppID, nil
	case "profile":
		return c.Profile, nil
	case "format":
		return c.Format, nil
	case "aws_region":
		return c.AWSRegion, nil
	case "open_browser_command":
		return c.OpenBrowserCommand, nil
	case "session_duration":
		return strconv.Itoa(c.SessionDuration), nil
	case "qr_code":
		return strconv.FormatBool(c.QRCode), nil
	case "open_browser":
		return strconv.FormatBool(c.OpenBrowser), nil
	case "all_profiles":
		return strconv.FormatBool(c.AllProfiles), nil
	case "write_aws_credentials":
		return strconv.FormatBool(c.WriteAWSCredentials), nil
	case "cache_access_token":
		return strconv.FormatBool(c.CacheAccessToken), nil
	case "debug":
		return strconv.FormatBool(c.Debug), nil
	case "debug_api_calls":
		return strconv.FormatBool(c.DebugAPICalls), nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}
func init() {
	viper.AutomaticEnv()
}
