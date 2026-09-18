// router.ts — minimal hash-based client router. No library: this is ~40
// lines and owning it directly avoids a dependency for something this
// small. Hash fragments are never sent to the server, so cmd/awis-server
// needs no server-side routing/fallback configuration regardless of how
// many client-side routes exist (GUI_PHASE1_ARCHITECTURE.md §1).

export type RouteParams = Record<string, string>;
export type RouteHandler = (params: RouteParams) => void;

interface CompiledRoute {
  pattern: RegExp;
  keys: string[];
  handler: RouteHandler;
}

const routes: CompiledRoute[] = [];
let notFoundRedirect = "/";

export function route(pathPattern: string, handler: RouteHandler): void {
  const keys: string[] = [];
  const regexSource = pathPattern.replace(/:[a-zA-Z_]+/g, (match) => {
    keys.push(match.slice(1));
    return "([^/]+)";
  });
  routes.push({ pattern: new RegExp(`^${regexSource}$`), keys, handler });
}

export function setNotFoundRedirect(path: string): void {
  notFoundRedirect = path;
}

function currentPath(): string {
  const h = location.hash.replace(/^#/, "");
  return h || "/";
}

function dispatch(): void {
  const path = currentPath();
  for (const r of routes) {
    const m = r.pattern.exec(path);
    if (m) {
      const params: RouteParams = {};
      r.keys.forEach((key, i) => {
        params[key] = decodeURIComponent(m[i + 1]!);
      });
      r.handler(params);
      return;
    }
  }
  location.hash = "#" + notFoundRedirect;
}

export function startRouter(): void {
  window.addEventListener("hashchange", dispatch);
  dispatch();
}

export function navigate(path: string): void {
  location.hash = "#" + path;
}
