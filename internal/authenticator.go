package internal

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"gopkg.in/ini.v1"
)

type Authenticator struct {
	config     *Config
	httpClient *http.Client
}

func NewAuthenticator(cfg *Config) *Authenticator {
	return &Authenticator{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *Authenticator) Authenticate() error {
	authFlow := a.config.AuthFlow
	if authFlow == "auto" {
		authFlow = a.detectAuthFlow()
	}

	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "Using authentication flow: %s\n", authFlow)
	}

	switch authFlow {
	case "oidc":
		return a.AuthenticateWithOIDC()
	case "saml-browser", "saml_browser":
		return a.AuthenticateWithBrowser()
	default:
		return fmt.Errorf("unknown authentication flow: %s (valid options: oidc, saml-browser, auto)", authFlow)
	}
}

func (a *Authenticator) detectAuthFlow() string {
	if a.config.OIDCClientID != "" {
		return "oidc"
	}
	if a.config.AWSAcctFedAppID != "" {
		return "saml-browser"
	}
	return "oidc"
}

func (a *Authenticator) AuthenticateWithOIDC() error {
	deviceAuth, err := a.startDeviceAuthorization()
	if err != nil {
		return fmt.Errorf("device authorization failed: %w", err)
	}

	if err := a.displayAuthorizationURL(deviceAuth); err != nil {
		return err
	}

	accessToken, err := a.pollForAccessToken(deviceAuth)
	if err != nil {
		return fmt.Errorf("failed to obtain access token: %w", err)
	}

	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ Access token obtained\n")
	}

	if a.config.CacheAccessToken {
		if err := a.cacheAccessToken(accessToken); err != nil && a.config.Debug {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache token: %v\n", err)
		}
	}

	appID := a.config.AWSAcctFedAppID
	if appID == "" {
		appID, err = a.discoverAWSFedApp(accessToken)
		if err != nil {
			return fmt.Errorf("failed to discover AWS Federation app: %w", err)
		}
	}

	samlAssertion, err := a.getSAMLAssertion(accessToken, appID)
	if err != nil {
		return fmt.Errorf("failed to get SAML assertion: %w", err)
	}

	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ SAML assertion obtained\n")
	}

	roles, err := a.extractRolesFromSAML(samlAssertion)
	if err != nil {
		return fmt.Errorf("failed to extract roles from SAML: %w", err)
	}

	roleARN, principalARN, err := a.selectRole(roles)
	if err != nil {
		return fmt.Errorf("failed to select role: %w", err)
	}

	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ Using role: %s\n", roleARN)
	}

	creds, err := a.assumeRoleWithSAML(samlAssertion, roleARN, principalARN)
	if err != nil {
		return fmt.Errorf("failed to assume role: %w", err)
	}

	return a.outputCredentials(creds)
}

type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func (a *Authenticator) startDeviceAuthorization() (*deviceAuthResponse, error) {
	authURL := fmt.Sprintf("https://%s/oauth2/v1/device/authorize", a.config.OrgDomain)

	data := url.Values{}
	data.Set("client_id", a.config.OIDCClientID)
	data.Set("scope", "openid profile okta.apps.sso")

	if a.config.DebugAPICalls {
		fmt.Fprintf(os.Stderr, "POST %s\n", authURL)
		fmt.Fprintf(os.Stderr, "Body: %s\n", data.Encode())
	}

	req, err := http.NewRequest("POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "oktaws/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.DebugAPICalls {
		fmt.Fprintf(os.Stderr, "Response: %d\n%s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceAuth deviceAuthResponse
	if err := json.Unmarshal(body, &deviceAuth); err != nil {
		return nil, fmt.Errorf("failed to parse device authorization response: %w", err)
	}

	return &deviceAuth, nil
}

func (a *Authenticator) displayAuthorizationURL(deviceAuth *deviceAuthResponse) error {
	fmt.Println()
	fmt.Println("To authenticate, visit:")
	fmt.Println()
	fmt.Printf("  %s\n", deviceAuth.VerificationURIComplete)
	fmt.Println()
	fmt.Printf("Or go to %s and enter code: %s\n", deviceAuth.VerificationURI, deviceAuth.UserCode)
	fmt.Println()

	if a.config.OpenBrowser {
		if err := a.openBrowser(deviceAuth.VerificationURIComplete); err != nil && a.config.Debug {
			fmt.Fprintf(os.Stderr, "Warning: failed to open browser: %v\n", err)
		}
	}

	return nil
}

func (a *Authenticator) openBrowser(url string) error {
	var cmd *exec.Cmd

	if a.config.OpenBrowserCommand != "" {
		cmd = exec.Command(a.config.OpenBrowserCommand, url)
	} else {
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return fmt.Errorf("unsupported platform for browser opening")
		}
	}

	return cmd.Start()
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func (a *Authenticator) pollForAccessToken(deviceAuth *deviceAuthResponse) (string, error) {
	tokenURL := fmt.Sprintf("https://%s/oauth2/v1/token", a.config.OrgDomain)

	interval := time.Duration(deviceAuth.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	timeout := time.After(time.Duration(deviceAuth.ExpiresIn) * time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Print("Waiting for authentication")

	for {
		select {
		case <-timeout:
			fmt.Println()
			return "", fmt.Errorf("authentication timed out")

		case <-ticker.C:
			fmt.Print(".")

			data := url.Values{}
			data.Set("client_id", a.config.OIDCClientID)
			data.Set("device_code", deviceAuth.DeviceCode)
			data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

			req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
			if err != nil {
				continue
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")

			resp, err := a.httpClient.Do(req)
			if err != nil {
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var tokenResp tokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				continue
			}

			if tokenResp.AccessToken != "" {
				fmt.Println(" ✓")
				return tokenResp.AccessToken, nil
			}

			if tokenResp.Error != "" && tokenResp.Error != "authorization_pending" && tokenResp.Error != "slow_down" {
				fmt.Println()
				return "", fmt.Errorf("authentication failed: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
			}
		}
	}
}

func (a *Authenticator) cacheAccessToken(token string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".okta", "awscli")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	cacheFile := filepath.Join(cacheDir, "cache.json")
	data := map[string]interface{}{
		"accessToken": token,
		"expiresAt":   time.Now().Add(1 * time.Hour).Unix(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, jsonData, 0600)
}

type appLink struct {
	AppName       string `json:"appName"`
	AppInstanceID string `json:"appInstanceId"`
	LinkURL       string `json:"linkUrl"`
	Label         string `json:"label"`
}

func (a *Authenticator) discoverAWSFedApp(accessToken string) (string, error) {
	appsURL := fmt.Sprintf("https://%s/api/v1/users/me/appLinks", a.config.OrgDomain)

	req, err := http.NewRequest("GET", appsURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to list apps: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apps []appLink
	if err := json.Unmarshal(body, &apps); err != nil {
		return "", fmt.Errorf("failed to parse apps response: %w", err)
	}

	for _, app := range apps {
		if strings.Contains(strings.ToLower(app.Label), "aws") && strings.Contains(app.LinkURL, "/app/amazon_aws/") {
			parts := strings.Split(app.LinkURL, "/")
			for i, part := range parts {
				if part == "amazon_aws" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no AWS Federation app found in Okta apps list")
}

func (a *Authenticator) getSAMLAssertion(accessToken, appID string) (string, error) {
	samlURL := fmt.Sprintf("https://%s/app/amazon_aws/%s/sso/saml", a.config.OrgDomain, appID)

	req, err := http.NewRequest("GET", samlURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "text/html")

	if a.config.DebugAPICalls {
		fmt.Fprintf(os.Stderr, "GET %s\n", samlURL)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if a.config.DebugAPICalls {
		fmt.Fprintf(os.Stderr, "Response: %d\n%s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SAML request failed with status %d", resp.StatusCode)
	}

	htmlContent := string(body)
	start := strings.Index(htmlContent, `name="SAMLResponse" value="`)
	if start == -1 {
		return "", fmt.Errorf("SAMLResponse not found in HTML")
	}

	start += len(`name="SAMLResponse" value="`)
	end := strings.Index(htmlContent[start:], `"`)
	if end == -1 {
		return "", fmt.Errorf("malformed SAMLResponse in HTML")
	}

	return htmlContent[start : start+end], nil
}

type samlResponse struct {
	XMLName   xml.Name `xml:"Response"`
	Assertion struct {
		AttributeStatement struct {
			Attributes []struct {
				Name           string `xml:"Name,attr"`
				AttributeValue []struct {
					Value string `xml:",chardata"`
				} `xml:"AttributeValue"`
			} `xml:"Attribute"`
		} `xml:"AttributeStatement"`
	} `xml:"Assertion"`
}

type awsRole struct {
	RoleARN      string
	PrincipalARN string
}

func (a *Authenticator) extractRolesFromSAML(samlAssertion string) ([]awsRole, error) {
	decodedSAML, err := base64.StdEncoding.DecodeString(samlAssertion)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAML: %w", err)
	}

	var samlResp samlResponse
	if err := xml.Unmarshal(decodedSAML, &samlResp); err != nil {
		return nil, fmt.Errorf("failed to parse SAML XML: %w", err)
	}

	var roles []awsRole
	for _, attr := range samlResp.Assertion.AttributeStatement.Attributes {
		if attr.Name == "https://aws.amazon.com/SAML/Attributes/Role" {
			for _, attrValue := range attr.AttributeValue {
				parts := strings.Split(attrValue.Value, ",")
				if len(parts) == 2 {
					var role awsRole
					if strings.Contains(parts[0], ":role/") {
						role.RoleARN = parts[0]
						role.PrincipalARN = parts[1]
					} else {
						role.RoleARN = parts[1]
						role.PrincipalARN = parts[0]
					}
					roles = append(roles, role)
				}
			}
		}
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf("no IAM roles found in SAML assertion")
	}

	return roles, nil
}

func (a *Authenticator) selectRole(roles []awsRole) (string, string, error) {
	if a.config.AWSIAMRole != "" {
		for _, role := range roles {
			if strings.Contains(role.RoleARN, a.config.AWSIAMRole) {
				return role.RoleARN, role.PrincipalARN, nil
			}
		}
		return "", "", fmt.Errorf("configured role %s not found in available roles", a.config.AWSIAMRole)
	}

	if len(roles) == 1 {
		return roles[0].RoleARN, roles[0].PrincipalARN, nil
	}

	fmt.Println("\nAvailable AWS roles:")
	for i, role := range roles {
		fmt.Printf("  [%d] %s\n", i+1, role.RoleARN)
	}

	fmt.Print("\nSelect a role [1]: ")
	var choice int
	fmt.Scanln(&choice)

	if choice == 0 {
		choice = 1
	}

	if choice < 1 || choice > len(roles) {
		return "", "", fmt.Errorf("invalid role selection")
	}

	selectedRole := roles[choice-1]
	return selectedRole.RoleARN, selectedRole.PrincipalARN, nil
}

func (a *Authenticator) assumeRoleWithSAML(samlAssertion, roleARN, principalARN string) (*sts.Credentials, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(a.config.AWSRegion),
	}))

	stsClient := sts.New(sess)

	input := &sts.AssumeRoleWithSAMLInput{
		RoleArn:         aws.String(roleARN),
		PrincipalArn:    aws.String(principalARN),
		SAMLAssertion:   aws.String(samlAssertion),
		DurationSeconds: aws.Int64(int64(a.config.SessionDuration)),
	}

	result, err := stsClient.AssumeRoleWithSAML(input)
	if err != nil {
		return nil, err
	}

	return result.Credentials, nil
}

func (a *Authenticator) outputCredentials(creds *sts.Credentials) error {
	if a.config.WriteAWSCredentials || a.config.Profile != "" {
		if err := a.writeCredentialsFile(creds); err != nil {
			return fmt.Errorf("failed to write credentials: %w", err)
		}
		log.Printf("Credentials written to profile: %s", a.config.Profile)
		return nil
	}

	switch a.config.Format {
	case "json":
		return a.outputJSON(creds)
	case "env":
		return a.outputEnv(creds)
	default:
		return a.outputJSON(creds)
	}
}

func (a *Authenticator) writeCredentialsFile(creds *sts.Credentials) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	awsDir := filepath.Join(homeDir, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return err
	}

	credsFile := filepath.Join(awsDir, "credentials")
	cfg, err := ini.Load(credsFile)
	if err != nil {
		cfg = ini.Empty()
	}

	profile := a.config.Profile
	if profile == "" {
		profile = "default"
	}

	section, err := cfg.NewSection(profile)
	if err != nil {
		section, _ = cfg.GetSection(profile)
	}

	section.Key("aws_access_key_id").SetValue(*creds.AccessKeyId)
	section.Key("aws_secret_access_key").SetValue(*creds.SecretAccessKey)
	section.Key("aws_session_token").SetValue(*creds.SessionToken)

	return cfg.SaveTo(credsFile)
}

func (a *Authenticator) outputJSON(creds *sts.Credentials) error {
	output := map[string]string{
		"AccessKeyId":     *creds.AccessKeyId,
		"SecretAccessKey": *creds.SecretAccessKey,
		"SessionToken":    *creds.SessionToken,
		"Expiration":      creds.Expiration.Format(time.RFC3339),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (a *Authenticator) outputEnv(creds *sts.Credentials) error {
	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", *creds.AccessKeyId)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", *creds.SecretAccessKey)
	fmt.Printf("export AWS_SESSION_TOKEN=%s\n", *creds.SessionToken)
	return nil
}
