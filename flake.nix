{
  description = "go-outside - Calculate if you should go outside based on current conditions.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable-small";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    nur.url = "github:nix-community/NUR";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    gomod2nix,
    nur,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [gomod2nix.overlays.default nur.overlays.default];
          config.allowUnfreePredicate = pkg: builtins.elem (pkgs.lib.getName pkg) ["goreleaser-pro"];
        };
        lib = pkgs.lib;
        version =
          if (self ? shortRev)
          then self.shortRev
          else "dev";
      in {
        packages.default = pkgs.buildGoApplication {
          pname = "go-outside";
          inherit version;

          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./gomod2nix.toml
              (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) ./.)
            ];
          };

          modules = ./gomod2nix.toml;

          subPackages = ["."];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/asphaltbuffet/go-outside/internal/versioncmd.Version=${version}"
          ];

          meta = with lib; {
            description = "Calculate if you should go outside based on current conditions.";
            homepage = "https://github.com/asphaltbuffet/go-outside";
            license = licenses.mit;
            mainProgram = "go-outside";
          };
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            jujutsu
            jjui
            mise
            ripgrep
            fd
            sd
            gopls
            nixd
            pkgs.nur.repos.goreleaser.goreleaser-pro
            gomod2nix.packages.${system}.default
            gh
          ];

          shellHook = ''
            mise trust --all
          '';
        };
      }
    )
    // {
      overlays.default = final: prev: {
        go-outside = self.packages.${prev.system}.default;
      };
    };
}
