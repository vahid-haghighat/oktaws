package cmd

import (
	"fmt"
	"oktaws/internal"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage oktaws configuration",
	Long:  `Manage oktaws configuration settings stored in ~/.config/oktaws/config.yaml`,
}
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration interactively",
	Long:  `Interactive setup wizard to configure oktaws`,
	RunE:  runConfigInit,
}
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a specific configuration value`,
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  `Get a specific configuration value`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration",
	Long:  `Display all configuration settings`,
	RunE:  runConfigList,
}
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long:  `Display the path to the configuration file`,
	RunE:  runConfigPath,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configPathCmd)
}
func runConfigInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Oktaws Configuration Setup")
	fmt.Println("==========================")
	fmt.Println()
	cfg := &internal.Config{}
	fmt.Println("Select authentication flow:")
	fmt.Println("  1. auto     - Auto-detect based on available configuration (recommended)")
	fmt.Println("  2. oidc     - OIDC device authorization flow")
	fmt.Println("  3. saml-browser - Browser-based SAML flow")
	fmt.Print("Choice [1]: ")
	var choice string
	fmt.Scanln(&choice)
	switch choice {
	case "2":
		cfg.AuthFlow = "oidc"
	case "3":
		cfg.AuthFlow = "saml-browser"
	default:
		cfg.AuthFlow = "auto"
	}
	fmt.Print("\nOkta organization domain (e.g., company.okta.com): ")
	fmt.Scanln(&cfg.OrgDomain)
	if cfg.AuthFlow == "oidc" || cfg.AuthFlow == "auto" {
		fmt.Print("OIDC Client ID (optional, press Enter to skip): ")
		fmt.Scanln(&cfg.OIDCClientID)
	}
	fmt.Print("AWS Account Federation App ID (e.g., exk123...): ")
	fmt.Scanln(&cfg.AWSAcctFedAppID)
	fmt.Print("AWS IAM Role ARN (optional, press Enter to skip): ")
	fmt.Scanln(&cfg.AWSIAMRole)
	fmt.Print("AWS Region [us-east-1]: ")
	fmt.Scanln(&cfg.AWSRegion)
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "us-east-1"
	}
	fmt.Print("AWS Profile name [default]: ")
	fmt.Scanln(&cfg.Profile)
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	fmt.Print("Session duration in seconds [3600]: ")
	var durationStr string
	fmt.Scanln(&durationStr)
	if durationStr == "" {
		cfg.SessionDuration = 3600
	} else {
		fmt.Sscanf(durationStr, "%d", &cfg.SessionDuration)
	}
	fmt.Print("Automatically open browser? [y/N]: ")
	var openBrowser string
	fmt.Scanln(&openBrowser)
	cfg.OpenBrowser = strings.ToLower(openBrowser) == "y" || strings.ToLower(openBrowser) == "yes"
	if err := cfg.SaveToFile(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	configPath := internal.GetConfigPath()
	fmt.Printf("\n✓ Configuration saved to: %s\n", configPath)
	return nil
}
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]
	cfg, err := internal.LoadConfigFromFile()
	if err != nil {
		cfg = &internal.Config{}
	}
	if err := cfg.SetValue(key, value); err != nil {
		return err
	}
	if err := cfg.SaveToFile(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	cfg, err := internal.LoadConfigFromFile()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	value, err := cfg.GetValue(key)
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}
func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := internal.LoadConfigFromFile()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	fmt.Println("Current Configuration:")
	fmt.Println("=====================")
	fmt.Printf("auth_flow:            %s\n", cfg.AuthFlow)
	fmt.Printf("org_domain:           %s\n", cfg.OrgDomain)
	fmt.Printf("oidc_client_id:       %s\n", cfg.OIDCClientID)
	fmt.Printf("aws_acct_fed_app_id:  %s\n", cfg.AWSAcctFedAppID)
	fmt.Printf("aws_iam_role:         %s\n", cfg.AWSIAMRole)
	fmt.Printf("aws_region:           %s\n", cfg.AWSRegion)
	fmt.Printf("profile:              %s\n", cfg.Profile)
	fmt.Printf("session_duration:     %d\n", cfg.SessionDuration)
	fmt.Printf("open_browser:         %t\n", cfg.OpenBrowser)
	fmt.Printf("debug:                %t\n", cfg.Debug)
	return nil
}
func runConfigPath(cmd *cobra.Command, args []string) error {
	configPath := internal.GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("%s (not created yet)\n", configPath)
	} else {
		fmt.Println(configPath)
	}
	return nil
}
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "oktaws"), nil
}
