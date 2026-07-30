{
  description = "snowstorm - Snowflake data-access CLI";

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
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        snowstorm = pkgs.buildGoModule {
          pname = "snowstorm";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-N3aDd7NxrNvJ+Fx17WZwrkF8O+TVFE7ZkabZDWe03II=";

          meta = with pkgs.lib; {
            description = "Snowflake data-access CLI: run queries, get structured JSON back";
            homepage = "https://github.com/bashfulrobot/snowstorm";
            license = licenses.mit;
            mainProgram = "snowstorm";
          };
        };
      in
      {
        packages = {
          default = snowstorm;
          snowstorm = snowstorm;
        };

        apps.default = flake-utils.lib.mkApp { drv = snowstorm; };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
            gofumpt
            delve
          ];

          shellHook = ''
            echo "snowstorm dev shell: $(go version)"
          '';
        };

        formatter = pkgs.nixfmt-tree;
      }
    )
    // {
      overlays.default = final: _prev: {
        snowstorm = self.packages.${final.stdenv.hostPlatform.system}.default;
      };
    };
}
