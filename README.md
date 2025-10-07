# oktaws

A CLI tool to obtain AWS credentials via Okta authentication, supporting both OIDC and SAML workflows.

## Features

- **OIDC Device Flow**: Authenticate using Okta OIDC with device authorization
- **SAML Browser Flow**: Seamless browser-based SAML authentication with automatic credential capture
- **Flexible Configuration**: Configure via YAML file, environment variables, or CLI flags
- **Multiple Output Formats**: Export credentials as JSON or environment variables
- **AWS Credentials File**: Automatically write to `~/.aws/credentials`
- **Browser Extension**: Auto-installs Chrome/Firefox extension for SAML interception

## Installation

```bash
go build -o oktaws
```

## Quick Start

### 1. Initialize Configuration

```bash
./oktaws config init
```

This creates `~/.config/oktaws/config.yaml` with default values.

### 2. Set Your Okta Configuration

```bash
./oktaws config set org_domain your-org.okta.com
./oktaws config set aws_acct_fed_app_id your-app-id
./oktaws config set aws_region us-east-1
```

### 3. Authenticate

```bash
./oktaws
```

On first run, the CLI will:
1. Detect your default browser (Chrome or Firefox)
2. Guide you through a one-time extension installation
3. Open your browser to Okta
4. Automatically capture SAML and fetch AWS credentials

## Configuration

### Configuration File

Location: `~/.config/oktaws/config.yaml`

```yaml
org_domain: your-org.okta.com
aws_acct_fed_app_id: exkXXXXXXXXXXXXXXXX
aws_region: us-east-1
session_duration: 43200  # 12 hours
auth_flow: saml-browser  # or "oidc" or "auto"
profile: default
format: env-var  # or "json"
```

### Configuration Commands

```bash
# View current configuration
./oktaws config list

# Get a specific value
./oktaws config get org_domain

# Set a value
./oktaws config set org_domain your-org.okta.com

# Show config file path
./oktaws config path
```

### Configuration Priority

1. CLI flags (highest priority)
2. Environment variables
3. Config file
4. Defaults (lowest priority)

## Authentication Flows

### SAML Browser Flow (Recommended)

```bash
./oktaws --auth-flow saml-browser
```

Best for:
- Users without Okta admin access
- Standard AWS/Okta SAML SSO setup
- Interactive workflows

**How it works:**
1. Starts a local callback server
2. Opens your browser to Okta
3. Browser extension intercepts SAML assertion
4. CLI receives SAML and calls AWS STS
5. Outputs AWS credentials

### OIDC Device Flow

```bash
./oktaws --auth-flow oidc --oidc-client-id your-client-id
```

Best for:
- CI/CD environments
- Headless servers
- When you have an OIDC client configured in Okta

**How it works:**
1. Generates a device code
2. Displays URL and code for user authorization
3. Polls Okta for access token
4. Exchanges token for SAML assertion
5. Calls AWS STS for credentials

## CLI Flags

### Authentication
- `--auth-flow string` - Authentication flow: `auto`, `oidc`, or `saml-browser` (default: auto)
- `--org-domain string` - Okta organization domain
- `--oidc-client-id string` - OIDC client ID (for OIDC flow)
- `--aws-acct-fed-app-id string` - AWS Account Federation app ID

### AWS Configuration
- `--aws-region string` - AWS region (default: us-east-1)
- `--aws-iam-role string` - AWS IAM role ARN (optional, will prompt if multiple)
- `--aws-session-duration string` - Session duration in seconds (default: 3600)

### Output
- `--format string` - Output format: `env-var` or `json` (default: env-var)
- `--profile string` - AWS profile name (default: default)
- `--write-aws-credentials` - Write to `~/.aws/credentials`

### Browser
- `--open-browser` - Open browser automatically (default: true for SAML flow)
- `--open-browser-command string` - Custom browser command

### Other
- `--debug` - Enable debug output
- `--debug-api-calls` - Debug API calls

## Output Formats

### Environment Variables (default)

```bash
export AWS_ACCESS_KEY_ID="ASIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_SESSION_TOKEN="..."
```

Use with:
```bash
eval $(./oktaws)
```

### JSON

```bash
./oktaws --format json
```

```json
{
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "...",
  "SessionToken": "...",
  "Expiration": "2025-10-07T07:50:15Z"
}
```

### AWS Credentials File

```bash
./oktaws --write-aws-credentials --profile my-profile
```

Writes to `~/.aws/credentials`:
```ini
[my-profile]
aws_access_key_id = ASIA...
aws_secret_access_key = ...
aws_session_token = ...
```

## Browser Extension

The SAML browser flow requires a browser extension that automatically captures SAML assertions.

### First-Time Setup

When you first run with `--auth-flow saml-browser`, the CLI will:

1. **Detect your browser** (Chrome or Firefox)
2. **Extract extension files** to `~/.config/oktaws/extension/`
3. **Enable Chrome Developer Mode** (if using Chrome)
4. **Open the extensions page** in your browser
5. **Guide you through installation** (one click: "Load unpacked")

### Supported Browsers

- Google Chrome
- Mozilla Firefox

Other browsers are not currently supported.

### Extension Permissions

The extension requires:
- Access to Okta domains (`*.okta.com`, `*.okta-emea.com`)
- Access to AWS signin (`*.signin.aws.amazon.com`)
- Network request interception (to capture SAML)
- Local storage (to save CLI port)

## Finding Your Configuration Values

### Okta Organization Domain

Your Okta domain is in the URL when you log into Okta:
```
https://your-org.okta.com
         ^^^^^^^^^^^^^^^^
```

### AWS Account Federation App ID

1. Log into Okta
2. Navigate to your AWS tile
3. Look at the URL:
```
https://your-org.okta.com/app/amazon_aws/exkXXXXXXXXXXXXXXXX/sso/saml
                                         ^^^^^^^^^^^^^^^^^^^^
```

### OIDC Client ID (if using OIDC flow)

Ask your Okta administrator to create an OIDC client for you and provide the client ID.

## Examples

### Example 1: Quick credentials for default profile

```bash
# Configure once
./oktaws config set org_domain my-company.okta.com
./oktaws config set aws_acct_fed_app_id exk123456789

# Get credentials
eval $(./oktaws)

# Use AWS CLI
aws s3 ls
```

### Example 2: Multiple AWS profiles

```bash
# Dev account
./oktaws --profile dev --aws-acct-fed-app-id exkDEV123 --write-aws-credentials

# Prod account  
./oktaws --profile prod --aws-acct-fed-app-id exkPROD456 --write-aws-credentials

# Use profiles
aws s3 ls --profile dev
aws s3 ls --profile prod
```

### Example 3: Long-lived session

```bash
./oktaws --aws-session-duration 43200  # 12 hours
```

### Example 4: Specific role selection

```bash
./oktaws --aws-iam-role admin-role
```

### Example 5: JSON output for scripting

```bash
CREDS=$(./oktaws --format json)
ACCESS_KEY=$(echo $CREDS | jq -r '.AccessKeyId')
echo "Access Key: $ACCESS_KEY"
```

## Troubleshooting

### Extension not detected

**Issue**: CLI says extension is not installed, but you installed it.

**Solution**: 
- Make sure Chrome is running
- Verify the extension is enabled in `chrome://extensions/`
- The extension name should be "Oktaws SAML Interceptor"
- Try restarting Chrome after installation

### Port already in use

**Issue**: `bind: address already in use` on port 8765

**Solution**:
```bash
# Kill the process using the port
lsof -ti:8765 | xargs kill -9
```

### SAML not captured

**Issue**: Browser opens but SAML is never received.

**Solution**:
- Check browser console for `[Oktaws]` messages
- Verify extension is loaded: `chrome://extensions/`
- Try reloading the extension
- Make sure you're authenticating (not already logged into AWS)

### Extension permissions error

**Issue**: Extension needs additional permissions.

**Solution**:
- Click the extension icon in your browser
- Grant the requested permissions
- Refresh the Okta page

### Multiple roles available

**Issue**: You have access to multiple AWS roles.

**Solution**:
- CLI will prompt you to select a role
- Or specify with `--aws-iam-role role-name`
- Add to config file to avoid prompts:
  ```yaml
  aws_iam_role: admin-role
  ```

### Browser doesn't open

**Issue**: CLI doesn't open browser automatically.

**Solution**:
- CLI will print the URL - open it manually
- Or set a custom browser: `--open-browser-command "/path/to/browser"`

## Security Notes

- **Credentials**: Never commit your config file with credentials to version control
- **Extensions**: The extension only runs on Okta and AWS domains
- **Local server**: The callback server only listens on localhost (127.0.0.1)
- **No data storage**: SAML assertions are not stored, only used in memory
- **Token caching**: Optional, disabled by default (`--cache-access-token` to enable)

## Contributing

Issues and pull requests are welcome!

## License

MIT


