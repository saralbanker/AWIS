// build.mjs — esbuild build script for the AWIS GUI (Phase 1).
//
// Bundles web/src/main.ts into cmd/awis-server/static/bundle.js and copies
// the static index.html/style.css alongside it. cmd/awis-server/static.go
// embeds that directory via go:embed, so this script's output location is
// not arbitrary — it is exactly what the shipped binary serves.
import * as esbuild from "esbuild";
import { copyFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const here = path.dirname(fileURLToPath(import.meta.url));
const outdir = path.join(here, "..", "cmd", "awis-server", "static");

mkdirSync(outdir, { recursive: true });

const watch = process.argv.includes("--watch");

const buildOptions = {
  entryPoints: [path.join(here, "src", "main.ts")],
  bundle: true,
  outfile: path.join(outdir, "bundle.js"),
  format: "esm",
  target: "es2022",
  sourcemap: true,
  logLevel: "info",
};

function copyStaticFiles() {
  copyFileSync(path.join(here, "index.html"), path.join(outdir, "index.html"));
  copyFileSync(path.join(here, "style.css"), path.join(outdir, "style.css"));
}

if (watch) {
  const ctx = await esbuild.context(buildOptions);
  copyStaticFiles();
  await ctx.watch();
  console.log("watching for changes...");
} else {
  await esbuild.build(buildOptions);
  copyStaticFiles();
  console.log(`built -> ${outdir}`);
}
