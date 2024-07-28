{
  description = "storage-console";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      type = "github";
      owner = "nix-community";
      repo = "gomod2nix";
      flake = true;
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

        devShell = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go_1_22
            golangci-lint
            gomod2nix.packages.${system}.default
            (mkGoEnv { pwd = ./.; })

            minio
          ];
        };
      }
    );
}

