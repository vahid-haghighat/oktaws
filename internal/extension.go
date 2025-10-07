package internal

import (
	"archive/zip"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed extension/*
var extensionFS embed.FS

type BrowserType string

const (
	BrowserChrome  BrowserType = "chrome"
	BrowserFirefox BrowserType = "firefox"
	BrowserUnknown BrowserType = "unknown"
)

func DetectDefaultBrowser() (BrowserType, string, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectBrowserMacOS()
	case "linux":
		return detectBrowserLinux()
	case "windows":
		return detectBrowserWindows()
	default:
		return BrowserUnknown, "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
func detectBrowserMacOS() (BrowserType, string, error) {
	cmd := exec.Command("defaults", "read", "com.apple.LaunchServices/com.apple.launchservices.secure", "LSHandlers")
	output, err := cmd.Output()
	if err != nil {
		return detectInstalledBrowserMacOS()
	}
	outputStr := string(output)
	if strings.Contains(outputStr, "com.google.chrome") {
		return BrowserChrome, "Google Chrome", nil
	}
	if strings.Contains(outputStr, "org.mozilla.firefox") {
		return BrowserFirefox, "Firefox", nil
	}
	return detectInstalledBrowserMacOS()
}
func detectInstalledBrowserMacOS() (BrowserType, string, error) {
	if _, err := os.Stat("/Applications/Google Chrome.app"); err == nil {
		return BrowserChrome, "Google Chrome", nil
	}
	if _, err := os.Stat("/Applications/Firefox.app"); err == nil {
		return BrowserFirefox, "Firefox", nil
	}
	return BrowserUnknown, "", fmt.Errorf("no supported browser found (Chrome or Firefox required)")
}
func detectBrowserLinux() (BrowserType, string, error) {
	cmd := exec.Command("xdg-settings", "get", "default-web-browser")
	output, err := cmd.Output()
	if err == nil {
		browser := strings.ToLower(strings.TrimSpace(string(output)))
		if strings.Contains(browser, "chrome") || strings.Contains(browser, "chromium") {
			return BrowserChrome, "Chrome", nil
		}
		if strings.Contains(browser, "firefox") {
			return BrowserFirefox, "Firefox", nil
		}
	}
	if _, err := exec.LookPath("google-chrome"); err == nil {
		return BrowserChrome, "Chrome", nil
	}
	if _, err := exec.LookPath("chromium"); err == nil {
		return BrowserChrome, "Chromium", nil
	}
	if _, err := exec.LookPath("firefox"); err == nil {
		return BrowserFirefox, "Firefox", nil
	}
	return BrowserUnknown, "", fmt.Errorf("no supported browser found (Chrome or Firefox required)")
}
func detectBrowserWindows() (BrowserType, string, error) {
	chromePath := filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe")
	if _, err := os.Stat(chromePath); err == nil {
		return BrowserChrome, "Google Chrome", nil
	}
	chromePath = filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe")
	if _, err := os.Stat(chromePath); err == nil {
		return BrowserChrome, "Google Chrome", nil
	}
	firefoxPath := filepath.Join(os.Getenv("ProgramFiles"), "Mozilla Firefox", "firefox.exe")
	if _, err := os.Stat(firefoxPath); err == nil {
		return BrowserFirefox, "Firefox", nil
	}
	firefoxPath = filepath.Join(os.Getenv("ProgramFiles(x86)"), "Mozilla Firefox", "firefox.exe")
	if _, err := os.Stat(firefoxPath); err == nil {
		return BrowserFirefox, "Firefox", nil
	}
	return BrowserUnknown, "", fmt.Errorf("no supported browser found (Chrome or Firefox required)")
}
func GetExtensionPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	extPath := filepath.Join(homeDir, ".config", "oktaws", "extension")

	if err := ensureExtensionExtracted(extPath); err != nil {
		return "", fmt.Errorf("failed to setup extension: %w", err)
	}

	return extPath, nil
}

func ensureExtensionExtracted(extPath string) error {
	manifestPath := filepath.Join(extPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(extPath, 0755); err != nil {
		return err
	}

	files := []string{
		"manifest.json",
		"background.js",
		"content.js",
		"popup.html",
		"popup.js",
		"icon16.png",
		"icon48.png",
		"icon128.png",
	}

	for _, file := range files {
		data, err := extensionFS.ReadFile("extension/" + file)
		if err != nil {
			return fmt.Errorf("failed to read embedded %s: %w", file, err)
		}

		destPath := filepath.Join(extPath, file)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	return nil
}
func InstallExtension(browserType BrowserType) error {
	extPath, err := GetExtensionPath()
	if err != nil {
		return fmt.Errorf("failed to get extension path: %w", err)
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Extension Setup Required (One-Time)                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	switch browserType {
	case BrowserChrome:
		return installChromeExtension(extPath)
	case BrowserFirefox:
		return installFirefoxExtension(extPath)
	default:
		return fmt.Errorf("unsupported browser type")
	}
}

func isChromeRunning() bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pgrep", "-x", "Google Chrome")
	case "linux":
		cmd = exec.Command("pgrep", "-x", "chrome")
	case "windows":
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe")
	default:
		return false
	}

	err := cmd.Run()
	return err == nil
}

func isFirefoxRunning() bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pgrep", "-x", "firefox")
	case "linux":
		cmd = exec.Command("pgrep", "-x", "firefox")
	case "windows":
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq firefox.exe")
	default:
		return false
	}

	err := cmd.Run()
	return err == nil
}

func openChromeExtensionsPage() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Google Chrome", "chrome://extensions/")
	case "linux":
		cmd = exec.Command("google-chrome", "chrome://extensions/")
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "chrome", "chrome://extensions/")
	default:
		return fmt.Errorf("unsupported OS")
	}
	return cmd.Start()
}

func openFirefoxDebuggingPage() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Firefox", "about:debugging#/runtime/this-firefox")
	case "linux":
		cmd = exec.Command("firefox", "about:debugging#/runtime/this-firefox")
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "firefox", "about:debugging#/runtime/this-firefox")
	default:
		return fmt.Errorf("unsupported OS")
	}
	return cmd.Start()
}

func openFolderInFinder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		return fmt.Errorf("unsupported OS")
	}
	return cmd.Start()
}

func LaunchBrowserWithExtension(browserType BrowserType, url string) error {
	extPath, err := GetExtensionPath()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	switch browserType {
	case BrowserChrome:
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				"--load-extension="+extPath,
				"--no-first-run",
				"--no-default-browser-check",
				url)
		case "linux":
			cmd = exec.Command("google-chrome",
				"--load-extension="+extPath,
				"--no-first-run",
				"--no-default-browser-check",
				url)
		case "windows":
			cmd = exec.Command("chrome.exe",
				"--load-extension="+extPath,
				"--no-first-run",
				"--no-default-browser-check",
				url)
		default:
			return fmt.Errorf("unsupported OS")
		}

	case BrowserFirefox:
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("/Applications/Firefox.app/Contents/MacOS/firefox",
				"-profile", getFirefoxProfilePath(),
				"-url", url)
		case "linux":
			cmd = exec.Command("firefox",
				"-profile", getFirefoxProfilePath(),
				"-url", url)
		case "windows":
			cmd = exec.Command("firefox.exe",
				"-profile", getFirefoxProfilePath(),
				"-url", url)
		default:
			return fmt.Errorf("unsupported OS")
		}

		if err := installFirefoxExtensionToProfile(extPath, getFirefoxProfilePath()); err != nil {
			return fmt.Errorf("failed to install Firefox extension: %w", err)
		}

	default:
		return fmt.Errorf("unsupported browser")
	}

	return cmd.Start()
}

func installChromeExtension(extPath string) error {
	if isChromeRunning() {
		fmt.Println("Chrome is currently running.")
		fmt.Print("Please close Chrome and press Enter to continue... ")
		var input string
		fmt.Scanln(&input)

		time.Sleep(1 * time.Second)

		if isChromeRunning() {
			fmt.Println("⚠ Chrome is still running. Waiting...")
			time.Sleep(2 * time.Second)
		}
	}

	homeDir, _ := os.UserHomeDir()
	var prefsPath string

	switch runtime.GOOS {
	case "darwin":
		prefsPath = filepath.Join(homeDir, "Library/Application Support/Google/Chrome/Default/Preferences")
	case "linux":
		prefsPath = filepath.Join(homeDir, ".config/google-chrome/Default/Preferences")
	case "windows":
		prefsPath = filepath.Join(homeDir, "AppData/Local/Google/Chrome/User Data/Default/Preferences")
	default:
		return fmt.Errorf("unsupported OS")
	}

	if err := enableChromeDevMode(prefsPath); err != nil {
		fmt.Println("⚠ Could not auto-enable Developer Mode")
		fmt.Println("  You'll need to enable it manually in the next step")
	} else {
		fmt.Println("✓ Developer Mode enabled")
	}

	fmt.Println()
	fmt.Println("Opening Chrome extensions page...")

	if err := openChromeExtensionsPage(); err != nil {
		return fmt.Errorf("failed to open Chrome: %w", err)
	}

	time.Sleep(2 * time.Second)

	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("✓ Extension folder ready at:")
	fmt.Println("  " + extPath)
	fmt.Println()
	fmt.Println("In the Chrome tab that just opened:")
	fmt.Println("  1. Developer Mode should already be ON (top-right)")
	fmt.Println("  2. Click 'Load unpacked' button")
	fmt.Println("  3. Select the folder path shown above")
	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("💡 Tip: Copy the path above, then paste it in the folder picker")
	fmt.Println()
	fmt.Print("Press Enter once the extension is loaded... ")

	var input string
	fmt.Scanln(&input)

	return nil
}

func enableChromeDevMode(prefsPath string) error {
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		return fmt.Errorf("Chrome preferences not found - please run Chrome at least once")
	}

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return err
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return err
	}

	if prefs["extensions"] == nil {
		prefs["extensions"] = make(map[string]interface{})
	}

	extensions := prefs["extensions"].(map[string]interface{})

	if extensions["ui"] == nil {
		extensions["ui"] = make(map[string]interface{})
	}

	ui := extensions["ui"].(map[string]interface{})
	ui["developer_mode"] = true

	updatedData, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(prefsPath, updatedData, 0644)
}

func installFirefoxExtension(extPath string) error {
	if isFirefoxRunning() {
		fmt.Println("Firefox is currently running.")
		fmt.Print("Please close Firefox and press Enter to continue... ")
		var input string
		fmt.Scanln(&input)

		time.Sleep(1 * time.Second)

		if isFirefoxRunning() {
			fmt.Println("⚠ Firefox is still running. Waiting...")
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println()
	fmt.Println("Opening Firefox debugging page...")

	if err := openFirefoxDebuggingPage(); err != nil {
		return fmt.Errorf("failed to open Firefox: %w", err)
	}

	time.Sleep(2 * time.Second)

	manifestPath := filepath.Join(extPath, "manifest.json")

	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("✓ Extension manifest ready at:")
	fmt.Println("  " + manifestPath)
	fmt.Println()
	fmt.Println("In the Firefox tab that just opened:")
	fmt.Println("  1. Click 'Load Temporary Add-on...' button")
	fmt.Println("  2. Select the manifest.json file from the path above")
	fmt.Println()
	fmt.Println("Note: This extension will need to be reloaded each time")
	fmt.Println("      Firefox restarts (browser security limitation)")
	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("💡 Tip: Copy the path above, then select manifest.json")
	fmt.Println()
	fmt.Print("Press Enter once the extension is loaded... ")

	var input string
	fmt.Scanln(&input)

	return nil
}

func installFirefoxExtensionToProfile(extPath, profilePath string) error {
	extensionsDir := filepath.Join(profilePath, "extensions")
	if err := os.MkdirAll(extensionsDir, 0755); err != nil {
		return err
	}

	extID := "oktaws@cli.extension"
	xpiPath := filepath.Join(extensionsDir, extID+".xpi")

	if _, err := os.Stat(xpiPath); err == nil {
		return nil
	}

	return createXPI(extPath, xpiPath)
}

func getFirefoxProfilePath() string {
	homeDir, _ := os.UserHomeDir()
	var profilesPath string

	switch runtime.GOOS {
	case "darwin":
		profilesPath = filepath.Join(homeDir, "Library/Application Support/Firefox/Profiles")
	case "linux":
		profilesPath = filepath.Join(homeDir, ".mozilla/firefox")
	case "windows":
		profilesPath = filepath.Join(homeDir, "AppData/Roaming/Mozilla/Firefox/Profiles")
	}

	profiles, err := filepath.Glob(filepath.Join(profilesPath, "*.default*"))
	if err != nil || len(profiles) == 0 {
		return profilesPath
	}

	return profiles[0]
}

func createXPI(sourceDir, destPath string) error {
	zipFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		fileContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = zipFileWriter.Write(fileContent)
		return err
	})
}

func CheckExtensionInstalled(browserType BrowserType) bool {
	ports := []int{8765}

	for _, port := range ports {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/status", port))
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return true
		}
	}

	return false
}

func IsExtensionInstalledInBrowser(browserType BrowserType) bool {
	extPath, err := GetExtensionPath()
	if err != nil {
		return false
	}

	manifestPath := filepath.Join(extPath, "manifest.json")
	_, err = os.Stat(manifestPath)
	return err == nil
}
