package main

// static.go — the GUI Phase 1 frontend's serving layer
// (docs/09-gui-planning/GUI_PHASE1_ARCHITECTURE.md §2).
//
// Production: staticFS embeds the built frontend (web/build.mjs's output,
// committed at cmd/awis-server/static/) directly into the binary — one
// process, no external files to ship alongside it, matching the rest of
// this codebase's deployment model.
//
// Development: --static-dir <path> (main.go) serves from disk instead, so a
// frontend engineer can run `npm run watch` against web/ and see edits on
// browser refresh with zero Go rebuild in the loop. This is a dev
// convenience only; the embedded path is the only thing a production build
// ships.

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/index.html static/style.css static/bundle.js static/bundle.js.map
var embeddedStatic embed.FS

// staticHandler returns an http.Handler serving the frontend's static
// assets at "/". When dir is non-empty, it serves directly from that
// directory on disk (dev mode); otherwise it serves the embedded build
// (production).
//
// Hash-based client-side routing (web/src/router.ts) means every
// client-side route still requests the same index.html — no server-side
// SPA-fallback logic is needed here, unlike History-API routing.
func staticHandler(dir string) (http.Handler, error) {
	if dir != "" {
		return http.FileServer(http.Dir(dir)), nil
	}
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}
