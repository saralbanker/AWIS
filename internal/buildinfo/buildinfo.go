// Package buildinfo holds the AWIS release string shared by every binary in
// this module (cmd/awis, cmd/awis-server). It exists so the two binaries
// cannot drift out of sync — cmd/awis-server needs this for GET
// /api/v1/info (GUI Beta, BE-2a/BE-2b) and previously had no way to see
// cmd/awis's version constant at all, since it lived unexported inside a
// different package main.
package buildinfo

// Version is the AWIS release string. "0.1.0-dev" is the development default.
// Release binaries set it at link time (.goreleaser.yaml passes
// -X github.com/awis/awis/internal/buildinfo.Version={{.Version}}), which only a
// package-level string variable accepts, so it must stay a var, not a const.
var Version = "0.1.0-dev"
