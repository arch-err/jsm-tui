# Configuration

jsm-tui requires a configuration file to connect to your Jira instance.

## Configuration File Location

The configuration file must be located at:

```
~/.config/jsm-tui/config.yaml
```

- **Linux/macOS**: `/home/username/.config/jsm-tui/config.yaml`
- **Windows**: `C:\Users\Username\.config\jsm-tui\config.yaml`

## Configuration Format

Create the configuration file in YAML format:

### Using Personal Access Token (PAT)

```yaml
url: https://your-jira-instance.com
auth:
  type: pat
  token: your-personal-access-token
project: YOUR-PROJECT-KEY
favorite_queues:
  - "Main"
  - "Assigned to me"
```

### Using Basic Authentication

```yaml
url: https://your-jira-instance.com
auth:
  type: basic
  username: your-username
  password: your-password
project: YOUR-PROJECT-KEY
favorite_queues:
  - "Main"
  - "Assigned to me"
```

## Configuration Fields

### `url` (required)

The base URL of your Jira Data Center instance.

- **Type**: string
- **Example**: `https://jira.company.com`
- Do not include trailing slash

### `auth` (required)

Authentication configuration object.

#### `auth.type` (required)

Authentication method to use.

- **Type**: string
- **Values**: `pat` or `basic`

#### `auth.token` (required for PAT)

Your Jira Personal Access Token.

- **Type**: string
- Required when `auth.type` is `pat`
- See [Creating a PAT](#creating-a-personal-access-token) below

#### `auth.username` (required for basic auth)

Your Jira username.

- **Type**: string
- Required when `auth.type` is `basic`

#### `auth.password` (required for basic auth)

Your Jira password.

- **Type**: string
- Required when `auth.type` is `basic`

### `project` (required)

The Service Desk project key.

- **Type**: string
- **Example**: `SD`, `SERVICEDESK`, `HELP`
- This is the project abbreviation shown in issue keys (e.g., `SD-123`)

### `favorite_queues` (optional)

List of queue names to mark as favorites. Favorite queues appear at the top of the queue list with a ★ indicator.

- **Type**: array of strings
- **Example**:
  ```yaml
  favorite_queues:
    - "Main"
    - "Assigned to me"
    - "High Priority"
  ```
- Queue names must match exactly as they appear in Jira (case-sensitive)
- If not specified, no queues will be marked as favorites

## Creating a Personal Access Token

Personal Access Tokens (PAT) are the recommended authentication method for Jira Data Center.

### Steps:

1. Log in to your Jira instance
2. Go to your profile settings
3. Navigate to **Personal Access Tokens**
4. Click **Create token**
5. Give it a meaningful name (e.g., "jsm-tui")
6. Set an expiration date or leave it unlimited
7. Click **Create**
8. Copy the token immediately (it won't be shown again)
9. Add the token to your `config.yaml`

!!! warning "Security Note"
    Never commit your configuration file with credentials to version control. Keep it secure and private.

## Example Configuration

Here's a complete example configuration:

```yaml
# Jira instance URL
url: https://jira.example.com

# Authentication (using PAT)
auth:
  type: pat
  token: ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop

# Service Desk project key
project: SD
```

## Troubleshooting

### Config file not found

If you see an error about the config file not being found:

1. Verify the file exists at `~/.config/jsm-tui/config.yaml`
2. Check file permissions (should be readable)
3. Ensure the directory `~/.config/jsm-tui/` exists

### Authentication failures

If you see authentication errors:

1. Verify your credentials are correct
2. Check that your PAT hasn't expired
3. Ensure your user has access to the Service Desk project
4. Verify the Jira URL is correct (no typos, correct protocol)

### Invalid project key

If you see errors about the project:

1. Verify the project key matches your Service Desk project
2. Ensure your user has access to view the project
3. Check that it's a Service Desk project (not a standard Jira project)

## Next Steps

After configuration, you're ready to [use jsm-tui](usage.md) to manage your Service Desk tickets.
