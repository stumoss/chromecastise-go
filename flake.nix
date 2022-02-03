{
  description = ''
    A simple command line application to transcode video files so that they are
    able to be played by chromecast devices.
  '';

  inputs.utils.url = "github:numtide/flake-utils";

  outputs = { self, utils, nixpkgs }:
    utils.lib.eachDefaultSystem (
      system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
          {
            packages.container = pkgs.dockerTools.buildLayeredImage {
              name = self.defaultPackage.${system}.pname;
              tag = self.defaultPackage.${system}.version;
              contents = [
                self.defaultPackage.${system}
              ];
              created = "now";

              config.Cmd = [ "chromecastise" ];
              includeStorePaths = false;
            };

            packages.streamContainer = pkgs.dockerTools.streamLayeredImage {
              name = self.defaultPackage.${system}.pname;
              tag = self.defaultPackage.${system}.version;
              contents = [
                self.defaultPackage.${system}
              ];

              config = {
                Cmd = [ "chromecastise" ];
              };
            };


            defaultPackage = pkgs.buildGoModule {
              pname = "chromecastise";
              version = "0.1.0";
              src = pkgs.lib.cleanSource ./.;
              nativeBuildInputs = with pkgs; [ makeWrapper ];

              postInstall = ''
                wrapProgram $out/bin/chromecastise --prefix PATH : ${pkgs.lib.strings.makeBinPath [ pkgs.mediainfo pkgs.ffmpeg ]}
              '';

              vendorSha256 = "sha256-+pMgaHB69itbQ+BDM7/oaJg3HrT1UN+joJL7BO/2vxE=";
            };

            devShell = with pkgs; pkgs.mkShell {
              name = "chromecastise-dev-shell";
              nativeBuildInputs = [
                go_1_17
              ];

              shellHook = ''
                export PS1="\e[0;31m[${self.defaultPackage.${system}.pname}]\$ \e[m"
                ${pkgs.cowsay}/bin/cowsay ${self.defaultPackage.${system}.pname}
              '';
            };
          }
    );
}
