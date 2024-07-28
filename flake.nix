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
  };

  outputs = { nixpkgs, flake-utils, gomod2nix, ... }:
    let
      utils = flake-utils;
    in
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        mkGoEnv = gomod2nix.legacyPackages.${system}.mkGoEnv;
      in
      {
        formatter = pkgs.nixpkgs-fmt;

        devShells = {
          default = pkgs.mkShell {
            nativeBuildInputs = with pkgs; [
              go_1_22
              golangci-lint
              gomod2nix.packages.${system}.default
              (mkGoEnv { pwd = ./.; })

              minio
            ];
          };
        };
      }
    );
}

