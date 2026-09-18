// Package buildinfo holds the AWIS release string shared by every binary in
// this module (cmd/awis, cmd/awis-server). It exists so the two binaries
// cannot drift out of sync — cmd/awis-server needs this for GET
// /api/v1/info (GUI Beta, BE-2a/BE-2b) and previously had no way to see
// cmd/awis's version constant at all, since it lived unexported inside a
// different package main.
package buildinfo

// Version is the AWIS release string.
// Freeze: update alongside go.mod module tag at release time.
const Version = "0.1.0-dev"
