{
  description = "storage-console";

  inputs = {
    nixpkgs = {
      type = "github";
      owner = "NixOS";
      repo = "nixpkgs";
      rev = "86e1ad4ec007f4f0e9561886935fe9b278860de8";
    };
    flake-utils = {
      type = "github";
      owner = "numtide";
      repo = "flake-utils";
      rev = "b1d9ab70662946ef0850d488da1c9019f3a9752a";
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
      ...
    }@inputs:
    let
      utils = flake-utils;
    in
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
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

        packages.default = pkgs.buildGoModule {
          pname = "storage-console";
          version = "0.3.0";
          vendorHash = "sha256-80QdvKdsIFkvYlgB0WomfmGC/gFD5iFX5nu0G6hO9mQ=";
          src = ./.;
          checkPhase = "";
          nativeBuildInputs = with pkgs; [ pkg-config ];
          buildInputs = with pkgs; [
            pkg-config
            vips
          ];
        };

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

              minio

              pkg-config
              vips

              dprint
              nixfmt-rfc-style
            ];
          };
        };
      }
    );
}
