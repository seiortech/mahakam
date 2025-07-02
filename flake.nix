{
  description = "Mahakam HTTP Framework development environment";

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
            go_1_24
            
            gomarkdoc
          ];

           shellHook = ''
            exec zsh
          '';
        };

        packages = {
          gen-docs = pkgs.writeShellScriptBin "gen-docs" ''
            echo "Generating documentation with gomarkdoc..."
            mkdir -p docs/packages
            
            echo "Generating docs for: mahakam (root)"
            ${pkgs.gomarkdoc}/bin/gomarkdoc --output "docs/packages/d1.mahakam.md" .
            sed -i '1i---\ntitle: Mahakam\nslug: mahakam\n---\n' docs/packages/mahakam.md
            
            for dir in */; do
              if [[ "$dir" == "example/" ]] || [[ "$dir" == "docs/" ]]; then
                echo "Skipping: $dir"
                continue
              fi
              
              if ls "$dir"*.go >/dev/null 2>&1; then
                pkg_name=$(basename "$dir")
                echo "Generating docs for: $pkg_name"
                ${pkgs.gomarkdoc}/bin/gomarkdoc --output "docs/packages/d1.$pkg_name.md" "./$dir"
                sed -i "1i---\ntitle: ''${pkg_name^}\nslug: $pkg_name\n---\n" "docs/packages/$pkg_name.md"
              fi
            done
            
            echo "Documentation generated in docs/packages/"
          '';
        };

        apps = {
          gen-docs = {
            type = "app";
            program = "${self.packages.${system}.gen-docs}/bin/gen-docs";
          };
        };
      }
    );
}