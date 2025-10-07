package internal

import (
	"fmt"
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

	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ Detected browser: %s\n", browserName)
	}

	fmt.Printf("✓ Detected browser: %s\n", browserName)

	server := NewCallbackServer(a.config)
	server.port = 8765
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer server.Shutdown()

	extInstalled := IsExtensionInstalledInBrowser(browserType)

	if !extInstalled {
		fmt.Println("⚠ Extension not detected in browser")
		fmt.Println()

		if err := InstallExtension(browserType); err != nil {
			return fmt.Errorf("failed to install extension: %w", err)
		}
	} else {
		fmt.Println("✓ Extension installed in browser")
	}

	oktaURL := fmt.Sprintf("https://%s/app/amazon_aws/%s/sso/saml",
		a.config.OrgDomain, a.config.AWSAcctFedAppID)

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  Okta Browser Authentication")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  ✓ Callback server running on port:", server.GetPort())
	fmt.Println("  ✓ Opening Okta in your browser...")
	fmt.Println()

	if err := a.openBrowser(oktaURL); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: Failed to open browser: %v\n", err)
		fmt.Printf("  Please manually open: %s\n\n", oktaURL)
	} else {
		if a.config.Debug {
			fmt.Fprintln(os.Stderr, "✓ Browser opened")
		}
	}
	fmt.Println("⏳ Waiting for authentication...")
	fmt.Println("   (The extension will automatically capture SAML)")
	fmt.Println()
	timeout := 5 * time.Minute
	samlAssertion, err := server.WaitForSAML(timeout)
	if err != nil {
		return fmt.Errorf("failed to receive SAML assertion: %w\n\nIf the extension didn't capture SAML, try:\n1. Refreshing the page\n2. Re-authenticating\n3. Checking that the extension is enabled", err)
	}
	fmt.Println("✓ SAML received from extension!")
	fmt.Println()
	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ SAML assertion received (%d bytes)\n", len(samlAssertion))
	}
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
	fmt.Println("✓ AWS credentials obtained!")
	fmt.Printf("  Role: %s\n", roleARN)
	fmt.Printf("  Expires: %s\n", credentials.Expiration.Format("2006-01-02 15:04:05"))
	fmt.Println()
	if a.config.Debug {
		fmt.Fprintf(os.Stderr, "✓ AWS credentials obtained\n")
		fmt.Fprintf(os.Stderr, "  Role: %s\n", roleARN)
		fmt.Fprintf(os.Stderr, "  Expiration: %s\n", credentials.Expiration.Format(time.RFC3339))
	}
	return a.outputCredentials(credentials)
}
