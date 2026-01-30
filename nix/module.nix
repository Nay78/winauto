{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.win-automation;

  # Generate JSON config from Nix options
  configFile = pkgs.writeText "win-automation.json" (
    builtins.toJSON {
      windows = {
        ssh_host = cfg.windows.sshHost;
        ssh_port = cfg.windows.sshPort;
        ssh_user = cfg.windows.sshUser;
        ssh_identity_file = cfg.windows.sshIdentityFile;
      };
      aloha = {
        server_url = cfg.aloha.serverUrl;
        client_url = cfg.aloha.clientUrl;
        server_start_cmd = cfg.aloha.serverStartCmd;
        client_start_cmd = cfg.aloha.clientStartCmd;
      };
      hatchet = {
        http_url = cfg.hatchet.httpUrl;
        grpc_address = cfg.hatchet.grpcAddress;
        health_url = cfg.hatchet.healthUrl;
        namespace = cfg.hatchet.namespace;
        worker_name = cfg.hatchet.workerName;
        worker_concurrency = cfg.hatchet.workerConcurrency;
        job_timeout = cfg.hatchet.jobTimeout;
        retry_max = cfg.hatchet.retryMax;
        retry_backoff = cfg.hatchet.retryBackoff;
      };
      playwright = {
        host = cfg.playwright.host;
        port = cfg.playwright.port;
      };
      artifacts = {
        out_dir = cfg.artifacts.outDir;
        retention_days = cfg.artifacts.retentionDays;
      };
      timeout = cfg.timeout;
    }
  );
in
{
  options.services.win-automation = {
    enable = lib.mkEnableOption "win-automation Windows VM orchestration";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.win-automation;
      defaultText = lib.literalExpression "pkgs.win-automation";
      description = "The win-automation package to use.";
    };

    # Windows SSH options
    windows = {
      sshHost = lib.mkOption {
        type = lib.types.str;
        default = "localhost";
        description = "Windows VM SSH host.";
      };

      sshPort = lib.mkOption {
        type = lib.types.port;
        default = 22555;
        description = "Windows VM SSH port.";
      };

      sshUser = lib.mkOption {
        type = lib.types.str;
        default = "administrator";
        description = "Windows VM SSH user.";
      };

      sshIdentityFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "Path to SSH identity file for Windows VM.";
      };
    };

    # Aloha options
    aloha = {
      serverUrl = lib.mkOption {
        type = lib.types.str;
        default = "http://127.0.0.1:7887";
        description = "Aloha server URL.";
      };

      clientUrl = lib.mkOption {
        type = lib.types.str;
        default = "http://127.0.0.1:7888";
        description = "Aloha client URL.";
      };

      serverStartCmd = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = "Command to start Aloha server on Windows.";
      };

      clientStartCmd = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = "Command to start Aloha client on Windows.";
      };
    };

    # Hatchet options
    hatchet = {
      httpUrl = lib.mkOption {
        type = lib.types.str;
        default = "http://127.0.0.1:8888";
        description = "Hatchet HTTP/UI URL.";
      };

      grpcAddress = lib.mkOption {
        type = lib.types.str;
        default = "localhost:7077";
        description = "Hatchet gRPC address.";
      };

      healthUrl = lib.mkOption {
        type = lib.types.str;
        default = "http://127.0.0.1:8733";
        description = "Hatchet health endpoint URL.";
      };

      tokenFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "Path to file containing Hatchet API token.";
      };

      namespace = lib.mkOption {
        type = lib.types.str;
        default = "default";
        description = "Hatchet namespace.";
      };

      workerName = lib.mkOption {
        type = lib.types.str;
        default = "win-automation";
        description = "Hatchet worker name.";
      };

      workerConcurrency = lib.mkOption {
        type = lib.types.int;
        default = 5;
        description = "Hatchet worker concurrency.";
      };

      jobTimeout = lib.mkOption {
        type = lib.types.str;
        default = "10m";
        description = "Default job timeout.";
      };

      retryMax = lib.mkOption {
        type = lib.types.int;
        default = 3;
        description = "Maximum retry attempts.";
      };

      retryBackoff = lib.mkOption {
        type = lib.types.str;
        default = "5s";
        description = "Retry backoff duration.";
      };
    };

    # Playwright options
    playwright = {
      host = lib.mkOption {
        type = lib.types.str;
        default = "127.0.0.1";
        description = "Playwright server host.";
      };

      port = lib.mkOption {
        type = lib.types.port;
        default = 9323;
        description = "Playwright server port.";
      };

      wsPathFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "Path to file containing Playwright WebSocket path secret.";
      };
    };

    # Artifacts options
    artifacts = {
      outDir = lib.mkOption {
        type = lib.types.path;
        default = "/var/lib/win-automation/artifacts";
        description = "Local directory for storing artifacts.";
      };

      retentionDays = lib.mkOption {
        type = lib.types.int;
        default = 7;
        description = "Artifact retention period in days.";
      };
    };

    # General options
    timeout = lib.mkOption {
      type = lib.types.str;
      default = "10s";
      description = "Default operation timeout.";
    };

    # Worker service options
    worker = {
      enable = lib.mkEnableOption "win-automation worker service";

      metrics = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Enable metrics emission.";
      };

      metricsInterval = lib.mkOption {
        type = lib.types.str;
        default = "30s";
        description = "Metrics emission interval.";
      };

      metricsPath = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "Path to write metrics. Defaults to stdout if not set.";
      };
    };

    # Supervisor service options
    supervisor = {
      enable = lib.mkEnableOption "win-automation supervisor service";

      interval = lib.mkOption {
        type = lib.types.str;
        default = "30s";
        description = "Health check interval.";
      };

      debug = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable verbose logging.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    # Ensure package is available
    environment.systemPackages = [ cfg.package ];

    # Create artifacts directory
    systemd.tmpfiles.rules = [
      "d ${cfg.artifacts.outDir} 0750 win-automation win-automation -"
    ];

    # Create system user
    users.users.win-automation = {
      isSystemUser = true;
      group = "win-automation";
      home = "/var/lib/win-automation";
      createHome = true;
      description = "win-automation service user";
    };

    users.groups.win-automation = { };

    # Worker service
    systemd.services.win-automation-worker = lib.mkIf cfg.worker.enable {
      description = "win-automation job worker";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        Type = "simple";
        User = "win-automation";
        Group = "win-automation";
        ExecStart = lib.concatStringsSep " " (
          [
            "${cfg.package}/bin/win-automation"
            "--config ${configFile}"
            "worker"
          ]
          ++ lib.optionals cfg.worker.metrics [
            "--metrics"
            "--metrics-interval ${cfg.worker.metricsInterval}"
          ]
        );
        Restart = "on-failure";
        RestartSec = "5s";

        # Security hardening
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        NoNewPrivileges = true;
        ReadWritePaths = [
          cfg.artifacts.outDir
          "/var/lib/win-automation"
        ];
      };

      environment = lib.mkMerge [
        {
          WIN_AUTOMATION_CONFIG = configFile;
        }
        (lib.mkIf (cfg.hatchet.tokenFile != null) {
          WIN_AUTOMATION_HATCHET_TOKEN_FILE = cfg.hatchet.tokenFile;
        })
        (lib.mkIf (cfg.playwright.wsPathFile != null) {
          WIN_AUTOMATION_PLAYWRIGHT_WS_PATH_FILE = cfg.playwright.wsPathFile;
        })
        (lib.mkIf (cfg.worker.metricsPath != null) {
          WIN_AUTOMATION_METRICS_PATH = cfg.worker.metricsPath;
        })
      ];
    };

    # Supervisor service
    systemd.services.win-automation-supervisor = lib.mkIf cfg.supervisor.enable {
      description = "win-automation session supervisor";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        Type = "simple";
        User = "win-automation";
        Group = "win-automation";
        ExecStart = lib.concatStringsSep " " (
          [
            "${cfg.package}/bin/win-automation"
            "--config ${configFile}"
            "supervisor"
            "run"
          ]
          ++ lib.optionals cfg.supervisor.debug [ "--debug" ]
        );
        Restart = "on-failure";
        RestartSec = "10s";

        # Security hardening
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        NoNewPrivileges = true;
        ReadWritePaths = [ "/var/lib/win-automation" ];
      };

      environment = lib.mkMerge [
        {
          WIN_AUTOMATION_CONFIG = configFile;
          WIN_AUTOMATION_SUPERVISOR_INTERVAL = cfg.supervisor.interval;
        }
        (lib.mkIf (cfg.hatchet.tokenFile != null) {
          WIN_AUTOMATION_HATCHET_TOKEN_FILE = cfg.hatchet.tokenFile;
        })
        (lib.mkIf (cfg.playwright.wsPathFile != null) {
          WIN_AUTOMATION_PLAYWRIGHT_WS_PATH_FILE = cfg.playwright.wsPathFile;
        })
      ];
    };
  };
}
