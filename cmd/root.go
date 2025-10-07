package cmd

import (
	"fmt"
	"oktaws/internal"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "oktaws",
	Short: "Okta AWS CLI - Web SSO authentication for AWS",
	Long: `Authenticate to AWS using Okta device authorization flow.
This tool performs OAuth device authorization with Okta, retrieves a SAML assertion,
and exchanges it for temporary AWS credentials.`,
	RunE: runWebAuth,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
func runWebAuth(cmd *cobra.Command, args []string) error {
	cfg, err := internal.NewConfig()
	if err != nil {
		return err
	}
	if cfg.OrgDomain == "" {
		return fmt.Errorf("org-domain is required (or set OKTA_AWSCLI_ORG_DOMAIN or run 'oktaws config init')")
	}
	authFlow := cfg.AuthFlow
	if authFlow == "auto" {
		if cfg.OIDCClientID != "" {
			authFlow = "oidc"
		} else if cfg.AWSAcctFedAppID != "" {
			authFlow = "saml-browser"
		}
	}
	if authFlow == "oidc" && cfg.OIDCClientID == "" {
		return fmt.Errorf("oidc-client-id is required for OIDC flow (or set OKTA_AWSCLI_OIDC_CLIENT_ID)")
	}
	if authFlow == "saml-browser" && cfg.AWSAcctFedAppID == "" {
		return fmt.Errorf("aws-acct-fed-app-id is required for browser SAML flow (run 'oktaws config init' to configure)")
	}
	auth := internal.NewAuthenticator(cfg)
	return auth.Authenticate()
}
func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.PersistentFlags().StringP("auth-flow", "x", "", "Authentication flow: auto, oidc, or saml-browser (default: auto)")
	rootCmd.PersistentFlags().StringP("org-domain", "o", os.Getenv("OKTA_AWSCLI_ORG_DOMAIN"), "Okta organization domain")
	rootCmd.PersistentFlags().StringP("oidc-client-id", "c", os.Getenv("OKTA_AWSCLI_OIDC_CLIENT_ID"), "OIDC client ID")
	rootCmd.PersistentFlags().StringP("aws-iam-role", "r", os.Getenv("OKTA_AWSCLI_IAM_ROLE"), "AWS IAM role ARN")
	rootCmd.PersistentFlags().StringP("aws-iam-idp", "i", os.Getenv("OKTA_AWSCLI_IAM_IDP"), "AWS IAM identity provider ARN")
	rootCmd.PersistentFlags().StringP("aws-acct-fed-app-id", "a", os.Getenv("OKTA_AWSCLI_AWS_ACCOUNT_FEDERATION_APP_ID"), "AWS Account Federation app ID")
	rootCmd.PersistentFlags().StringP("profile", "p", os.Getenv("OKTA_AWSCLI_PROFILE"), "AWS profile name")
	rootCmd.PersistentFlags().StringP("aws-session-duration", "s", os.Getenv("OKTA_AWSCLI_SESSION_DURATION"), "Session duration")
	rootCmd.PersistentFlags().StringP("format", "f", os.Getenv("OKTA_AWSCLI_FORMAT"), "Output format")
	rootCmd.PersistentFlags().StringP("aws-region", "n", os.Getenv("OKTA_AWSCLI_AWS_REGION"), "AWS region")
	rootCmd.PersistentFlags().BoolP("qr-code", "q", false, "Display QR code")
	rootCmd.PersistentFlags().BoolP("open-browser", "b", false, "Open browser automatically")
	rootCmd.PersistentFlags().StringP("open-browser-command", "m", os.Getenv("OKTA_AWSCLI_BROWSER_COMMAND"), "Browser command")
	rootCmd.PersistentFlags().BoolP("all-profiles", "k", false, "Collect all profiles")
	rootCmd.PersistentFlags().BoolP("write-aws-credentials", "w", false, "Write to ~/.aws/credentials")
	rootCmd.PersistentFlags().BoolP("cache-access-token", "e", false, "Cache access token")
	rootCmd.PersistentFlags().BoolP("debug", "g", false, "Debug mode")
	rootCmd.PersistentFlags().BoolP("debug-api-calls", "d", false, "Debug API calls")
	viper.BindPFlag("auth-flow", rootCmd.PersistentFlags().Lookup("auth-flow"))
	viper.BindPFlag("org-domain", rootCmd.PersistentFlags().Lookup("org-domain"))
	viper.BindPFlag("oidc-client-id", rootCmd.PersistentFlags().Lookup("oidc-client-id"))
	viper.BindPFlag("aws-iam-role", rootCmd.PersistentFlags().Lookup("aws-iam-role"))
	viper.BindPFlag("aws-iam-idp", rootCmd.PersistentFlags().Lookup("aws-iam-idp"))
	viper.BindPFlag("aws-acct-fed-app-id", rootCmd.PersistentFlags().Lookup("aws-acct-fed-app-id"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("aws-session-duration", rootCmd.PersistentFlags().Lookup("aws-session-duration"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	viper.BindPFlag("aws-region", rootCmd.PersistentFlags().Lookup("aws-region"))
	viper.BindPFlag("qr-code", rootCmd.PersistentFlags().Lookup("qr-code"))
	viper.BindPFlag("open-browser", rootCmd.PersistentFlags().Lookup("open-browser"))
	viper.BindPFlag("open-browser-command", rootCmd.PersistentFlags().Lookup("open-browser-command"))
	viper.BindPFlag("all-profiles", rootCmd.PersistentFlags().Lookup("all-profiles"))
	viper.BindPFlag("write-aws-credentials", rootCmd.PersistentFlags().Lookup("write-aws-credentials"))
	viper.BindPFlag("cache-access-token", rootCmd.PersistentFlags().Lookup("cache-access-token"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("debug-api-calls", rootCmd.PersistentFlags().Lookup("debug-api-calls"))
}
