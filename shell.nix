{ pkgs ? import <nixpkgs> {} }:

with pkgs;

mkShell {
  buildInputs = [
    go
    goimports
    # gotests
    delve
    golangci-lint
    errcheck
    go-check
    goconst
    gocyclo
    golint

    mediainfo
  ];
}
