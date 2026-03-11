# NOTE : SOURCE CODE FOR THE PATCHMON AGENT NOW LIVES IN THE MAIN PATCHMON REPO UNDER agent-source-code

# This Repo is now archived.

# PatchMon Agent

PatchMon's monitoring agent sends package and repository information to the PatchMon server. Supports **Linux** and **Windows**.

## Installation

### One-Line Install (Recommended)

The easiest way to install is using the one-liner from the PatchMon UI. After adding a host, copy the install command shown in the UI.

**Linux:**
```bash
curl -s http://your-server:3000/api/v1/hosts/install -H "X-API-ID: your_api_id" -H "X-API-KEY: your_api_key" | sh
```

**Windows (PowerShell as Administrator):**
```powershell
$script = Invoke-WebRequest -Uri "http://your-server:3000/api/v1/hosts/install" -Headers @{"X-API-ID"="your_api_id"; "X-API-KEY"="your_api_key"} -UseBasicParsing; $script.Content | Out-File -FilePath "$env:TEMP\patchmon-install.ps1" -Encoding utf8; powershell.exe -ExecutionPolicy Bypass -File "$env:TEMP\patchmon-install.ps1"
```

### Manual Binary Installation

#### Linux

1. **Download** the appropriate binary for your architecture from the releases
2. **Make executable** and move to system path:
   ```bash
   chmod +x patchmon-agent-linux-amd64
   sudo mv patchmon-agent-linux-amd64 /usr/local/bin/patchmon-agent
   ```

#### Windows

1. **Download** `patchmon-agent-windows-amd64.exe` from releases
2. **Move** to Program Files:
   ```powershell
   New-Item -ItemType Directory -Force -Path "C:\Program Files\PatchMon"
   Move-Item patchmon-agent-windows-amd64.exe "C:\Program Files\PatchMon\patchmon-agent.exe"
   ```
3. **Create Windows Service:**
   ```powershell
   New-Service -Name PatchMonAgent -BinaryPathName '"C:\Program Files\PatchMon\patchmon-agent.exe" serve' -DisplayName "PatchMon Agent" -StartupType Automatic
   Start-Service -Name PatchMonAgent
   ```

### From Source

1. **Prerequisites**:
   - Go 1.25 or later
   - Root access on the target system

2. **Build and Install**:
   ```bash
   # Clone or copy the source code
   make deps          # Install dependencies
   make build         # Build the application
   sudo make install  # Install to /usr/local/bin
   ```

## Configuration

### Initial Setup

1. **Configure Credentials**:
   ```bash
   sudo patchmon-agent config set-api <API_ID> <API_KEY> <SERVER_URL>
   ```

   Example:
   ```bash
   sudo patchmon-agent config set-api patchmon_1a2b3c4d abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890 http://patchmon.example.com
   ```

2. **Test Configuration**:
   ```bash
   sudo patchmon-agent ping
   ```

3. **Send Initial Report**:
   ```bash
   sudo patchmon-agent report
   ```

### Configuration Files

#### Linux
- **Main Config**: `/etc/patchmon/config.yml`
- **Credentials**: `/etc/patchmon/credentials.yml` (600 permissions)
- **Logs**: `/etc/patchmon/logs/patchmon-agent.log`

#### Windows
- **Main Config**: `C:\ProgramData\PatchMon\config.yml`
- **Credentials**: `C:\ProgramData\PatchMon\credentials.yml`
- **Logs**: `C:\ProgramData\PatchMon\patchmon-agent.log`
- **Binary**: `C:\Program Files\PatchMon\patchmon-agent.exe`

## Usage

### Available Commands

```bash
# Configuration and setup
sudo patchmon-agent config set-api <API_ID> <API_KEY> <SERVER_URL>  # Configure credentials
sudo patchmon-agent config show                                     # Show current config
sudo patchmon-agent ping                                            # Test credentials and connectivity

# Data collection and reporting
sudo patchmon-agent report                                          # Report system & package status to server

# Agent management
sudo patchmon-agent check-version                                   # Check for updates
sudo patchmon-agent update-agent                                    # Update to latest version
sudo patchmon-agent update-crontab                                  # Update cron schedule
sudo patchmon-agent uninstall [flags]                               # Uninstall the agent

# Diagnostics
sudo patchmon-agent diagnostics                                     # Show system diagnostics
```

### Example Configuration File

Create `/etc/patchmon/config.yml`:

```yaml
patchmon_server: "https://patchmon.example.com"
api_version: "v1"
credentials_file: "/etc/patchmon/credentials.yml"
log_file: "/var/log/patchmon-agent.log"
log_level: "info"
```

### Example Credentials File

The credentials file is automatically created by the `configure` command:

```yaml
api_id: "patchmon_1a2b3c4d5e6f7890"
api_key: "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
```

## Service Management

### Linux (systemd)

```bash
# Check status
systemctl status patchmon-agent

# Start/stop/restart
sudo systemctl start patchmon-agent
sudo systemctl stop patchmon-agent
sudo systemctl restart patchmon-agent

# View logs
journalctl -u patchmon-agent -f
```

### Windows Service

```powershell
# Check status
Get-Service -Name PatchMonAgent

# Start/stop/restart
Start-Service -Name PatchMonAgent
Stop-Service -Name PatchMonAgent
Restart-Service -Name PatchMonAgent

# View logs
Get-Content "C:\ProgramData\PatchMon\patchmon-agent.log" -Tail 50
```

## Automation

### Linux Crontab Setup

The agent can automatically configure crontab based on server policies:

```bash
# Update crontab with current server policy
sudo patchmon-agent update-crontab
```

This creates entries like:
```bash
# Hourly reports (at minute 15)
15 * * * * /usr/local/bin/patchmon-agent report >/dev/null 2>&1
15 * * * * /usr/local/bin/patchmon-agent update-crontab >/dev/null 2>&1
```

### Windows

On Windows, the agent runs as a Windows Service and handles scheduling internally. No crontab setup is needed.

## Uninstallation

### Linux

The agent includes a built-in uninstall command:

```bash
# Basic uninstall (removes binary, crontab, backups)
sudo patchmon-agent uninstall

# Complete uninstall (includes config and logs)
sudo patchmon-agent uninstall --remove-all

# Silent complete removal
sudo patchmon-agent uninstall -af
```

**Or use the server-provided script:**
```bash
curl -s http://your-server:3000/api/v1/hosts/remove | sh
```

### Windows

**One-liner removal (PowerShell as Administrator):**
```powershell
$script = Invoke-WebRequest -Uri "http://your-server:3000/api/v1/hosts/remove" -UseBasicParsing; $script.Content | Out-File -FilePath "$env:TEMP\patchmon-remove.ps1" -Encoding utf8; powershell.exe -ExecutionPolicy Bypass -File "$env:TEMP\patchmon-remove.ps1" -RemoveAll
```

**Manual removal:**
```powershell
# Stop and remove the service
Stop-Service -Name PatchMonAgent -Force
sc.exe delete PatchMonAgent

# Remove files
Remove-Item -Path "C:\Program Files\PatchMon" -Recurse -Force
Remove-Item -Path "C:\ProgramData\PatchMon" -Recurse -Force
```

### Uninstall Options
```
--remove-config    # Remove configuration and credentials files
--remove-logs      # Remove log files  
--remove-all, -a   # Remove all files (shortcut for --remove-config --remove-logs)
--force, -f        # Skip confirmation prompts
```

## Logging

Logs are written to `/var/log/patchmon-agent.log` with timestamps and structured format:

```
2023-09-27T10:30:00 level=info msg="Collecting package information..."
2023-09-27T10:30:01 level=info msg="Found packages" count=156
2023-09-27T10:30:02 level=info msg="Sending report to PatchMon server..."
2023-09-27T10:30:03 level=info msg="Report sent successfully"
```

Log levels: `debug`, `info`, `warn`, `error`

## Troubleshooting

### Common Issues

1. **Permission Denied**:
   ```bash
   # Ensure running as root
   sudo patchmon-agent <command>
   ```

2. **Credentials Not Found**:
   ```bash
   # Configure credentials first
   sudo patchmon-agent config set-api <API_ID> <API_KEY> <SERVER_URL>
   ```

3. **Network Connectivity**:
   ```bash
   # Test server connectivity
   sudo patchmon-agent ping
   sudo patchmon-agent diagnostics  # Detailed network info
   ```

4. **Package Manager Issues**:
   ```bash
   # Update package lists manually
   sudo apt update         # Ubuntu/Debian
   sudo dnf check-update   # Fedora/RHEL
   ```

### Diagnostics

Run comprehensive diagnostics:

```bash
sudo patchmon-agent diagnostics
```

This returns information about your system, the agent, the current configuration and server connectivity state, and more.

## Migration from Shell Script

The Go implementation maintains compatibility with the existing shell script workflow:

1. **Same command structure**: All commands work identically
2. **Same configuration files**: Uses the same paths and formats
3. **Same API compatibility**: Works with existing PatchMon servers
4. **Improved performance**: Faster execution and better error handling

To migrate:
1. Remove the old shell script agent, config, credentials, and crontab.
2. Install the Go binary as described above
3. No changes needed to crontab or server settings

## Development

### Building

```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for all supported platforms (Linux amd64, arm64, i386)
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format and lint
make fmt
make lint

# Clean build artifacts
make clean
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make fmt` and `make lint`
6. Submit a pull request
