{
  description = "CLI for automating a Windows VM from Linux via SSH and HTTP APIs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      # Systems to build for
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      # Version from git or "dev"
      version = if self ? rev then "git-${builtins.substring 0 7 self.rev}" else "dev";
    in
    flake-utils.lib.eachSystem supportedSystems (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        win-automation = pkgs.callPackage ./nix/package.nix {
          inherit version;
          src = self;
        };
      in
      {
        packages = {
          default = win-automation;
          win-automation = win-automation;
        };

        apps.default = flake-utils.lib.mkApp {
          drv = win-automation;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            just
          ];

          shellHook = ''
            echo "win-automation development shell"
            echo "Commands: just build, just test, just fmt"
          '';
        };
      }
    )
    // {
      # NixOS module
      nixosModules = {
        default = import ./nix/module.nix;
        win-automation = import ./nix/module.nix;
      };

      # Overlay to add win-automation to pkgs
      overlays.default = final: prev: {
        win-automation = final.callPackage ./nix/package.nix {
          version = if self ? rev then "git-${builtins.substring 0 7 self.rev}" else "dev";
          src = self;
        };
      };
    };
}
