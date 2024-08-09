{
  description = "storage-console";

  inputs = {
    nixpkgs = {
      type = "github";
      owner = "NixOS";
      repo = "nixpkgs";
      rev = "4cf7951a91440879f61e05460441762d59adc017";
    };
    flake-utils = {
      type = "github";
      owner = "numtide";
      repo = "flake-utils";
      rev = "b1d9ab70662946ef0850d488da1c9019f3a9752a";
    };
    gomod2nix = {
      type = "github";
      owner = "nix-community";
      repo = "gomod2nix";
      rev = "31b6d2e40b36456e792cd6cf50d5a8ddd2fa59a1";
    };
    pre-commit-hooks = {
      type = "github";
      owner = "cachix";
      repo = "git-hooks.nix";
      rev = "c7012d0c18567c889b948781bc74a501e92275d1";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      gomod2nix,
      ...
    }@inputs:
    let
      utils = flake-utils;
    in
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        mkGoEnv = gomod2nix.legacyPackages.${system}.mkGoEnv;
        goEnv = mkGoEnv { pwd = ./.; };
      in
      {
        checks = {
          pre-commit-check = inputs.pre-commit-hooks.lib.${system}.run {
            src = ./.;
            hooks = {
              dprint = {
                enable = true;
                name = "dprint check";
                entry = "dprint check --allow-no-files";
              };
              nixfmt = {
                enable = true;
                name = "nixfmt check";
                entry = "nixfmt -c ";
                types = [ "nix" ];
              };
            };
          };
        };

        formatter = pkgs.nixpkgs-fmt;

        devShells = {
          default = pkgs.mkShell {
            inherit (self.checks.${system}.pre-commit-check) shellHook;
            buildInputs = self.checks.${system}.pre-commit-check.enabledPackages;

            env = {
              DOCKER_HOST = "unix:///Users/charlieegan3/.colima/default/docker.sock";
              TESTCONTAINERS_RYUK_DISABLED = "true";
            };

            packages = with pkgs; [
              go_1_22
              golangci-lint
              gomod2nix.packages.${system}.default
              goEnv

              minio

              dprint
              nixfmt-rfc-style
            ];
          };
        };
      }
    );
}
