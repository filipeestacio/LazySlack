{
  description = "LazySlack - Terminal UI for Slack";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "lazyslack";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
        };
      });
}
