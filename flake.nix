{
  description = "liber - a small CLI bookmark manager";

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
        packages.default = pkgs.buildGoModule {
          pname = "liber";
          version = "0.5.1";
          src = ./.;

          # No third-party Go modules are used, so there's nothing to vendor.
          vendorHash = null;

          ldflags = [ "-X main.Version=0.5.1" ];

          # Optional runtime dependencies:
          #   - single-file-cli for `liber -a` archiving
          #   - fzf for the fuzzy picker in `liber -s` (falls back to a
          #     plain prompt without it, or use `liber -sl` to force that)
          # Uncomment if you want them wired into PATH automatically:
          # nativeBuildInputs = [ pkgs.makeWrapper ];
          # postInstall = ''
          #   wrapProgram $out/bin/liber --prefix PATH : ${pkgs.lib.makeBinPath [
          #     pkgs.nodePackages.single-file-cli
          #     pkgs.fzf
          #   ]}
          # '';

          meta = with pkgs.lib; {
            description = "A small CLI bookmark manager (html + markdown + archive)";
            homepage = "https://example.invalid/liber";
            license = licenses.mit;
            mainProgram = "liber";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go pkgs.gopls pkgs.fzf ];
        };

        apps.default = flake-utils.lib.mkApp { drv = self.packages.${system}.default; };
      });
}
