# NixOS Integration Notes

This repo is intended to run on the Linux host and talk to Windows via SSH + local port forwards.

## Flake Installation

### Quick Start

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    win-automation.url = "github:alejg/win-automation";
  };

  outputs = { self, nixpkgs, win-automation, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        win-automation.nixosModules.default
        ./configuration.nix
      ];
    };
  };
}
```

### Overlay (add to pkgs)

```nix
{
  nixpkgs.overlays = [ win-automation.overlays.default ];
  environment.systemPackages = [ pkgs.win-automation ];
}
```

### CLI Only (no services)

```bash
# Run directly
nix run github:alejg/win-automation -- doctor

# Install to profile
nix profile install github:alejg/win-automation
```

### Development Shell

```bash
nix develop github:alejg/win-automation
```

## NixOS Module Configuration

### Minimal (worker only)

```nix
{
  services.win-automation = {
    enable = true;
    worker.enable = true;
  };
}
```

### Full Configuration

```nix
{
  services.win-automation = {
    enable = true;

    # Windows VM SSH
    windows = {
      sshHost = "localhost";
      sshPort = 22555;
      sshUser = "administrator";
      sshIdentityFile = "/etc/win-automation/id_ed25519";
    };

    # Aloha GUI automation
    aloha = {
      serverUrl = "http://127.0.0.1:7887";
      clientUrl = "http://127.0.0.1:7888";
      serverStartCmd = "Start-Process -FilePath 'C:\\Aloha\\server.exe'";
      clientStartCmd = "Start-Process -FilePath 'C:\\Aloha\\client.exe'";
    };

    # Hatchet job queue
    hatchet = {
      httpUrl = "http://127.0.0.1:8888";
      grpcAddress = "localhost:7077";
      healthUrl = "http://127.0.0.1:8733";
      tokenFile = "/run/secrets/hatchet-token";
      namespace = "production";
      workerConcurrency = 10;
    };

    # Playwright browser automation
    playwright = {
      host = "127.0.0.1";
      port = 9323;
      wsPathFile = "/run/secrets/playwright-ws-path";
    };

    # Artifacts
    artifacts = {
      outDir = "/var/lib/win-automation/artifacts";
      retentionDays = 14;
    };

    # Worker service
    worker = {
      enable = true;
      metrics = true;
      metricsInterval = "30s";
      metricsPath = "/var/log/win-automation/metrics.log";
    };

    # Supervisor service (auto-repair)
    supervisor = {
      enable = true;
      interval = "30s";
      debug = false;
    };
  };
}
```

### Secrets Management (sops-nix example)

```nix
{
  sops.secrets = {
    hatchet-token = {
      sopsFile = ./secrets.yaml;
      owner = "win-automation";
    };
    playwright-ws-path = {
      sopsFile = ./secrets.yaml;
      owner = "win-automation";
    };
  };

  services.win-automation = {
    enable = true;
    hatchet.tokenFile = config.sops.secrets.hatchet-token.path;
    playwright.wsPathFile = config.sops.secrets.playwright-ws-path.path;
    worker.enable = true;
  };
}
```

### Service Management

```bash
# Check status
systemctl status win-automation-worker
systemctl status win-automation-supervisor

# View logs
journalctl -u win-automation-worker -f
journalctl -u win-automation-supervisor -f

# Restart
systemctl restart win-automation-worker

# Health check
win-automation doctor
```

## Building the Package

### Local Development Build

```bash
nix build .#win-automation
./result/bin/win-automation version
```

### Updating Dependencies

When Go dependencies change:

```bash
# Update vendor directory
go mod tidy
go mod vendor

# Commit the changes
git add go.mod go.sum vendor/
git commit -m "chore: update dependencies"
```

The flake uses `vendorHash = null` with the committed vendor directory, so no hash updates are needed.

---

In the current setup, the NixOS config repo wires:

- Windows VM (QEMU/KVM) + SSH forwarding: `modules/system/windows-vm.nix`
  - hostfwd binds to loopback: `127.0.0.1:<hostPort> -> guest:<guestPort>`

- Host-specific wiring (mini): `hosts/mini/configuration.nix`
  - `services.windowsServer.extraPortForwards` includes ports for:
    - 22555 -> 22 (SSH into Windows)
    - 7887 -> 7887 (Aloha server)
    - 7888 -> 7888 (Aloha client)

- Aloha file deployment + helper CLI: `hosts/mini/deployments/aloha.nix`
  - systemd `aloha-setup` copies PowerShell scripts into the virtio-fs share:
    `/var/lib/windows-server/shared/aloha` -> `Z:\aloha`

This project should remain usable outside that exact environment, but those are the reference paths when debugging.

## Hatchet-lite Integration

### Port Forwarding

Add Hatchet-lite ports to `services.windowsServer.extraPortForwards`:

```nix
{
  services.windowsServer.extraPortForwards = [
    { hostPort = 22555; guestPort = 22; }    # SSH
    { hostPort = 7887; guestPort = 7887; }   # Aloha server
    { hostPort = 7888; guestPort = 7888; }   # Aloha client
    { hostPort = 8888; guestPort = 8888; }   # Hatchet UI/HTTP
    { hostPort = 7077; guestPort = 7077; }   # Hatchet gRPC
    { hostPort = 8733; guestPort = 8733; }   # Hatchet health
  ];
}
```

### NixOS Service Declaration

Example systemd service for Hatchet-lite on the Linux host:

```nix
{
  systemd.services.hatchet-lite = {
    description = "Hatchet-lite task queue";
    after = [ "network.target" "postgresql.service" ];
    wantedBy = [ "multi-user.target" ];
    
    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.hatchet}/bin/hatchet server start";
      Restart = "on-failure";
      RestartSec = "5s";
      
      # Security hardening
      DynamicUser = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = true;
      NoNewPrivileges = true;
    };
    
    environment = {
      HATCHET_DATABASE_URL = "postgres://hatchet:hatchet@localhost:5432/hatchet";
      HATCHET_SERVER_PORT = "8888";
      HATCHET_GRPC_PORT = "7077";
      HATCHET_HEALTH_PORT = "8733";
    };
  };
}
```

### Compatibility Notes

**SDK Version Compatibility:**
- This project uses Hatchet SDK v0.77.36 (see `go.mod`)
- Compatible with Hatchet-lite v0.77.x
- Major.minor versions must match between SDK and server
- Patch versions are interchangeable within the same minor release

**Version Pinning in Nix:**
```nix
{
  # Pin Hatchet package to specific version
  hatchet = pkgs.hatchet.overrideAttrs (old: {
    version = "0.77.36";
    src = pkgs.fetchFromGitHub {
      owner = "hatchet-dev";
      repo = "hatchet";
      rev = "v0.77.36";
      sha256 = "...";
    };
  });
}
```

**Upgrade Strategy:**
1. Check compatibility matrix in `docs/CONTEXT.md`
2. Update SDK version in `go.mod`
3. Update Hatchet-lite package in NixOS config
4. Rebuild both: `just build` and `nixos-rebuild switch`
5. Verify with `win-automation doctor`

**Known Issues:**
- Mismatched major.minor versions will cause gRPC errors
- SDK v0.78+ requires Hatchet-lite v0.78+ (breaking changes)
- Health endpoints may return 503 during database migrations

## Ecosystem Compatibility Matrix

| Component | Linux Host | Windows VM | Version Constraint |
|-----------|------------|------------|-------------------|
| win-automation | Required | - | Latest |
| SSH | Required | OpenSSH Server | Any |
| Hatchet-lite | Required | - | v0.77.x |
| Hatchet SDK | Required | - | v0.77.x (match server) |
| Aloha Server | - | Required | Latest |
| Aloha Client | - | Required | Latest |
| Playwright | Optional | Required | Major/minor match |
| Node.js | - | Required (for Playwright) | LTS |

## Port Forwarding Requirements

| Service | Host Port | Guest Port | Protocol |
|---------|-----------|------------|----------|
| SSH | 22555 | 22 | TCP |
| Aloha Server | 7887 | 7887 | TCP |
| Aloha Client | 7888 | 7888 | TCP |
| Playwright | 9323 | 9323 | TCP |
| Hatchet HTTP | 8888 | 8888 | TCP |
| Hatchet gRPC | 7077 | 7077 | TCP |
| Hatchet Health | 8733 | 8733 | TCP |

## Drift Checklist

Run this checklist periodically to detect configuration drift:

### Network
- [ ] SSH port forwarding active (22555 → 22)
- [ ] Aloha ports forwarded (7887, 7888)
- [ ] Playwright port forwarded (9323)
- [ ] Hatchet ports forwarded (8888, 7077, 8733)

### Windows VM
- [ ] OpenSSH Server running
- [ ] Windows Firewall rules for Aloha (7887, 7888)
- [ ] Windows Firewall rules for Playwright (9323)
- [ ] Aloha server process running
- [ ] Aloha client process running
- [ ] Playwright scheduled task exists and running

### Linux Host
- [ ] Hatchet-lite running and healthy
- [ ] win-automation binary at expected version
- [ ] Config file present and valid

### Automated Check
```bash
# Run full health check
win-automation doctor

# Run supervisor for auto-repair
win-automation supervisor run --once
```
