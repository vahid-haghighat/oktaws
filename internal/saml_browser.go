package internal

import (
	"fmt"
	"log"
	"os"
	"time"
)

func (a *Authenticator) AuthenticateWithBrowser() error {
	if a.config.OrgDomain == "" {
		return fmt.Errorf("org-domain is required for browser authentication")
	}
	if a.config.AWSAcctFedAppID == "" {
		return fmt.Errorf("aws-acct-fed-app-id is required for browser authentication")
	}

	browserType, browserName, err := DetectDefaultBrowser()
	if err != nil {
		return fmt.Errorf("browser detection failed: %w\n\nSupported browsers: Chrome, Firefox", err)
	}
	if browserType == BrowserUnknown {
		return fmt.Errorf("unsupported browser. Please use Chrome or Firefox")
	}

	log.Printf("Detected browser: %s", browserName)

	server := NewCallbackServer(a.config)
	server.port = 8765
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer server.Shutdown()

	extInstalled := IsExtensionInstalledInBrowser(browserType)

	if !extInstalled {
		log.Printf("Extension not detected. Installing...")
		if err := InstallExtension(browserType); err != nil {
			return fmt.Errorf("failed to install extension: %w", err)
		}
	}

	oktaURL := fmt.Sprintf("https://%s/app/amazon_aws/%s/sso/saml",
		a.config.OrgDomain, a.config.AWSAcctFedAppID)

	log.Printf("Opening browser to Okta...")
	if err := a.openBrowser(oktaURL); err != nil {
		log.Printf("Warning: Failed to open browser: %v", err)
		log.Printf("Please manually open: %s", oktaURL)
	}
	timeout := 5 * time.Minute
	samlAssertion, err := server.WaitForSAML(timeout)
	if err != nil {
		return fmt.Errorf("failed to receive SAML assertion: %w\n\nIf the extension didn't capture SAML, try:\n1. Refreshing the page\n2. Re-authenticating\n3. Checking that the extension is enabled", err)
	}
	log.Printf("SAML assertion received (%d bytes)", len(samlAssertion))
	roles, err := a.extractRolesFromSAML(samlAssertion)
	if err != nil {
		return fmt.Errorf("failed to extract roles from SAML: %w", err)
	}
	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ Found %d role(s)\n", len(roles))
	}
	roleARN, principalARN, err := a.selectRole(roles)
	if err != nil {
		return fmt.Errorf("failed to select role: %w", err)
	}
	credentials, err := a.assumeRoleWithSAML(samlAssertion, roleARN, principalARN)
	if err != nil {
		return fmt.Errorf("failed to assume role: %w", err)
	}
	log.Printf("Retrieved credentials for account %s successfully", "AWS")
	log.Printf("Assumed role: %s", roleARN)
	log.Printf("Credentials expire at: %s", credentials.Expiration.Format("2006-01-02 15:04:05 -0700 MST"))
	return a.outputCredentials(credentials)
}
