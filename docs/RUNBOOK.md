# Operational Runbook

## Deployment

### Prerequisites
- Go 1.21+ installed
- SSH access to Windows VM configured
- Hatchet-lite running (see docs/CONTEXT.md)

### Build Release
```bash
# Build with version
VERSION=v1.0.0
go build -ldflags "-X main.version=$VERSION" -o win-automation ./cmd/win-automation

# Verify
./win-automation version
```

### Deploy Binary
```bash
# Copy to target host
scp win-automation user@host:/usr/local/bin/

# Verify deployment
ssh user@host "win-automation version"
```

### Start Worker
```bash
# Run worker with metrics
win-automation worker --metrics --metrics-interval 30s
```

## Upgrade

1. Build new version
2. Stop existing worker (graceful shutdown via SIGTERM)
3. Deploy new binary
4. Start worker
5. Verify with `win-automation doctor`

## Rollback

1. Stop current worker
2. Restore previous binary from backup
3. Start worker
4. Verify with `win-automation doctor`

## Troubleshooting

### SSH Connection Failed
```bash
# Check SSH connectivity
win-automation doctor
# Verify Windows firewall rules
win-automation supervisor run --once --debug
```

### Aloha Not Responding
```bash
# Check Aloha health
win-automation aloha health
# Run supervisor to auto-repair
win-automation supervisor run --once
```

### Hatchet Jobs Stuck
```bash
# Check Hatchet health
win-automation doctor
# Check job status
win-automation jobs status --id <job_id>
# Cancel stuck job
win-automation jobs cancel --id <job_id>
```

## Systemd Unit

Example systemd unit for Linux host:

```ini
[Unit]
Description=win-automation worker
After=network.target

[Service]
Type=simple
User=win-automation
ExecStart=/usr/local/bin/win-automation worker --metrics
Restart=always
RestartSec=5
Environment=WIN_AUTOMATION_CONFIG=/etc/win-automation/config.json
Environment=WIN_AUTOMATION_HATCHET_TOKEN=<token>

[Install]
WantedBy=multi-user.target
```

## Monitoring

### Health Checks
```bash
# Full system health check
win-automation doctor

# Component-specific checks
win-automation aloha health
win-automation windows exec -- "echo OK"
```

### Metrics
Worker exposes metrics when started with `--metrics`:
- Job execution counts
- Success/failure rates
- Execution duration
- SSH connection health
- Aloha availability

### Logs
```bash
# View worker logs (systemd)
journalctl -u win-automation -f

# View with structured output
win-automation worker --log-format json
```

## Configuration

### Environment Variables
- `WIN_AUTOMATION_CONFIG`: Path to config file (default: `./config.json`)
- `WIN_AUTOMATION_HATCHET_TOKEN`: Hatchet API token
- `WIN_AUTOMATION_HATCHET_URL`: Hatchet server URL (default: `http://localhost:7077`)
- `WIN_AUTOMATION_SSH_HOST`: Windows VM SSH host (default: `localhost:22555`)
- `WIN_AUTOMATION_ALOHA_SERVER`: Aloha server URL (default: `http://127.0.0.1:7887`)
- `WIN_AUTOMATION_ALOHA_CLIENT`: Aloha client URL (default: `http://127.0.0.1:7888`)

### Config File Format
```json
{
  "hatchet": {
    "url": "http://localhost:7077",
    "token": "<token>"
  },
  "ssh": {
    "host": "localhost:22555",
    "user": "Administrator",
    "timeout": "30s"
  },
  "aloha": {
    "server": "http://127.0.0.1:7887",
    "client": "http://127.0.0.1:7888",
    "timeout": "60s"
  }
}
```

## Security

### Secrets Management
- Never commit tokens or passwords to git
- Use environment variables or secret managers
- Rotate Hatchet tokens regularly
- Use SSH key authentication (not passwords)

### Network Security
- Restrict SSH access to known hosts
- Use firewall rules to limit Aloha exposure
- Run worker as unprivileged user
- Use TLS for Hatchet communication in production

## Backup and Recovery

### Configuration Backup
```bash
# Backup config
cp /etc/win-automation/config.json /backup/config.json.$(date +%Y%m%d)
```

### Binary Backup
```bash
# Before upgrade, backup current binary
cp /usr/local/bin/win-automation /backup/win-automation.$(date +%Y%m%d)
```

### Recovery
```bash
# Restore from backup
cp /backup/win-automation.YYYYMMDD /usr/local/bin/win-automation
systemctl restart win-automation
```
