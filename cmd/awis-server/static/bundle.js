// node_modules/lit-html/lit-html.js
var t = globalThis;
var i = (t4) => t4;
var s = t.trustedTypes;
var e = s ? s.createPolicy("lit-html", { createHTML: (t4) => t4 }) : void 0;
var h = "$lit$";
var o = `lit$${Math.random().toFixed(9).slice(2)}$`;
var n = "?" + o;
var r = `<${n}>`;
var l = document;
var c = () => l.createComment("");
var a = (t4) => null === t4 || "object" != typeof t4 && "function" != typeof t4;
var u = Array.isArray;
var d = (t4) => u(t4) || "function" == typeof t4?.[Symbol.iterator];
var f = "[ 	\n\f\r]";
var v = /<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g;
var _ = /-->/g;
var m = />/g;
var p = RegExp(`>|${f}(?:([^\\s"'>=/]+)(${f}*=${f}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`, "g");
var g = /'/g;
var $ = /"/g;
var y = /^(?:script|style|textarea|title)$/i;
var x = (t4) => (i4, ...s3) => ({ _$litType$: t4, strings: i4, values: s3 });
var b = x(1);
var w = x(2);
var T = x(3);
var E = /* @__PURE__ */ Symbol.for("lit-noChange");
var A = /* @__PURE__ */ Symbol.for("lit-nothing");
var C = /* @__PURE__ */ new WeakMap();
var P = l.createTreeWalker(l, 129);
function V(t4, i4) {
  if (!u(t4) || !t4.hasOwnProperty("raw")) throw Error("invalid template strings array");
  return void 0 !== e ? e.createHTML(i4) : i4;
}
var N = (t4, i4) => {
  const s3 = t4.length - 1, e3 = [];
  let n2, l2 = 2 === i4 ? "<svg>" : 3 === i4 ? "<math>" : "", c3 = v;
  for (let i5 = 0; i5 < s3; i5++) {
    const s4 = t4[i5];
    let a2, u4, d2 = -1, f2 = 0;
    for (; f2 < s4.length && (c3.lastIndex = f2, u4 = c3.exec(s4), null !== u4); ) f2 = c3.lastIndex, c3 === v ? "!--" === u4[1] ? c3 = _ : void 0 !== u4[1] ? c3 = m : void 0 !== u4[2] ? (y.test(u4[2]) && (n2 = RegExp("</" + u4[2], "g")), c3 = p) : void 0 !== u4[3] && (c3 = p) : c3 === p ? ">" === u4[0] ? (c3 = n2 ?? v, d2 = -1) : void 0 === u4[1] ? d2 = -2 : (d2 = c3.lastIndex - u4[2].length, a2 = u4[1], c3 = void 0 === u4[3] ? p : '"' === u4[3] ? $ : g) : c3 === $ || c3 === g ? c3 = p : c3 === _ || c3 === m ? c3 = v : (c3 = p, n2 = void 0);
    const x2 = c3 === p && t4[i5 + 1].startsWith("/>") ? " " : "";
    l2 += c3 === v ? s4 + r : d2 >= 0 ? (e3.push(a2), s4.slice(0, d2) + h + s4.slice(d2) + o + x2) : s4 + o + (-2 === d2 ? i5 : x2);
  }
  return [V(t4, l2 + (t4[s3] || "<?>") + (2 === i4 ? "</svg>" : 3 === i4 ? "</math>" : "")), e3];
};
var S = class _S {
  constructor({ strings: t4, _$litType$: i4 }, e3) {
    let r2;
    this.parts = [];
    let l2 = 0, a2 = 0;
    const u4 = t4.length - 1, d2 = this.parts, [f2, v3] = N(t4, i4);
    if (this.el = _S.createElement(f2, e3), P.currentNode = this.el.content, 2 === i4 || 3 === i4) {
      const t5 = this.el.content.firstChild;
      t5.replaceWith(...t5.childNodes);
    }
    for (; null !== (r2 = P.nextNode()) && d2.length < u4; ) {
      if (1 === r2.nodeType) {
        if (r2.hasAttributes()) for (const t5 of r2.getAttributeNames()) if (t5.endsWith(h)) {
          const i5 = v3[a2++], s3 = r2.getAttribute(t5).split(o), e4 = /([.?@])?(.*)/.exec(i5);
          d2.push({ type: 1, index: l2, name: e4[2], strings: s3, ctor: "." === e4[1] ? I : "?" === e4[1] ? L : "@" === e4[1] ? z : H }), r2.removeAttribute(t5);
        } else t5.startsWith(o) && (d2.push({ type: 6, index: l2 }), r2.removeAttribute(t5));
        if (y.test(r2.tagName)) {
          const t5 = r2.textContent.split(o), i5 = t5.length - 1;
          if (i5 > 0) {
            r2.textContent = s ? s.emptyScript : "";
            for (let s3 = 0; s3 < i5; s3++) r2.append(t5[s3], c()), P.nextNode(), d2.push({ type: 2, index: ++l2 });
            r2.append(t5[i5], c());
          }
        }
      } else if (8 === r2.nodeType) if (r2.data === n) d2.push({ type: 2, index: l2 });
      else {
        let t5 = -1;
        for (; -1 !== (t5 = r2.data.indexOf(o, t5 + 1)); ) d2.push({ type: 7, index: l2 }), t5 += o.length - 1;
      }
      l2++;
    }
  }
  static createElement(t4, i4) {
    const s3 = l.createElement("template");
    return s3.innerHTML = t4, s3;
  }
};
function M(t4, i4, s3 = t4, e3) {
  if (i4 === E) return i4;
  let h3 = void 0 !== e3 ? s3._$Co?.[e3] : s3._$Cl;
  const o2 = a(i4) ? void 0 : i4._$litDirective$;
  return h3?.constructor !== o2 && (h3?._$AO?.(false), void 0 === o2 ? h3 = void 0 : (h3 = new o2(t4), h3._$AT(t4, s3, e3)), void 0 !== e3 ? (s3._$Co ??= [])[e3] = h3 : s3._$Cl = h3), void 0 !== h3 && (i4 = M(t4, h3._$AS(t4, i4.values), h3, e3)), i4;
}
var R = class {
  constructor(t4, i4) {
    this._$AV = [], this._$AN = void 0, this._$AD = t4, this._$AM = i4;
  }
  get parentNode() {
    return this._$AM.parentNode;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  u(t4) {
    const { el: { content: i4 }, parts: s3 } = this._$AD, e3 = (t4?.creationScope ?? l).importNode(i4, true);
    P.currentNode = e3;
    let h3 = P.nextNode(), o2 = 0, n2 = 0, r2 = s3[0];
    for (; void 0 !== r2; ) {
      if (o2 === r2.index) {
        let i5;
        2 === r2.type ? i5 = new k(h3, h3.nextSibling, this, t4) : 1 === r2.type ? i5 = new r2.ctor(h3, r2.name, r2.strings, this, t4) : 6 === r2.type && (i5 = new Z(h3, this, t4)), this._$AV.push(i5), r2 = s3[++n2];
      }
      o2 !== r2?.index && (h3 = P.nextNode(), o2++);
    }
    return P.currentNode = l, e3;
  }
  p(t4) {
    let i4 = 0;
    for (const s3 of this._$AV) void 0 !== s3 && (void 0 !== s3.strings ? (s3._$AI(t4, s3, i4), i4 += s3.strings.length - 2) : s3._$AI(t4[i4])), i4++;
  }
};
var k = class _k {
  get _$AU() {
    return this._$AM?._$AU ?? this._$Cv;
  }
  constructor(t4, i4, s3, e3) {
    this.type = 2, this._$AH = A, this._$AN = void 0, this._$AA = t4, this._$AB = i4, this._$AM = s3, this.options = e3, this._$Cv = e3?.isConnected ?? true;
  }
  get parentNode() {
    let t4 = this._$AA.parentNode;
    const i4 = this._$AM;
    return void 0 !== i4 && 11 === t4?.nodeType && (t4 = i4.parentNode), t4;
  }
  get startNode() {
    return this._$AA;
  }
  get endNode() {
    return this._$AB;
  }
  _$AI(t4, i4 = this) {
    t4 = M(this, t4, i4), a(t4) ? t4 === A || null == t4 || "" === t4 ? (this._$AH !== A && this._$AR(), this._$AH = A) : t4 !== this._$AH && t4 !== E && this._(t4) : void 0 !== t4._$litType$ ? this.$(t4) : void 0 !== t4.nodeType ? this.T(t4) : d(t4) ? this.k(t4) : this._(t4);
  }
  O(t4) {
    return this._$AA.parentNode.insertBefore(t4, this._$AB);
  }
  T(t4) {
    this._$AH !== t4 && (this._$AR(), this._$AH = this.O(t4));
  }
  _(t4) {
    this._$AH !== A && a(this._$AH) ? this._$AA.nextSibling.data = t4 : this.T(l.createTextNode(t4)), this._$AH = t4;
  }
  $(t4) {
    const { values: i4, _$litType$: s3 } = t4, e3 = "number" == typeof s3 ? this._$AC(t4) : (void 0 === s3.el && (s3.el = S.createElement(V(s3.h, s3.h[0]), this.options)), s3);
    if (this._$AH?._$AD === e3) this._$AH.p(i4);
    else {
      const t5 = new R(e3, this), s4 = t5.u(this.options);
      t5.p(i4), this.T(s4), this._$AH = t5;
    }
  }
  _$AC(t4) {
    let i4 = C.get(t4.strings);
    return void 0 === i4 && C.set(t4.strings, i4 = new S(t4)), i4;
  }
  k(t4) {
    u(this._$AH) || (this._$AH = [], this._$AR());
    const i4 = this._$AH;
    let s3, e3 = 0;
    for (const h3 of t4) e3 === i4.length ? i4.push(s3 = new _k(this.O(c()), this.O(c()), this, this.options)) : s3 = i4[e3], s3._$AI(h3), e3++;
    e3 < i4.length && (this._$AR(s3 && s3._$AB.nextSibling, e3), i4.length = e3);
  }
  _$AR(t4 = this._$AA.nextSibling, s3) {
    for (this._$AP?.(false, true, s3); t4 !== this._$AB; ) {
      const s4 = i(t4).nextSibling;
      i(t4).remove(), t4 = s4;
    }
  }
  setConnected(t4) {
    void 0 === this._$AM && (this._$Cv = t4, this._$AP?.(t4));
  }
};
var H = class {
  get tagName() {
    return this.element.tagName;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  constructor(t4, i4, s3, e3, h3) {
    this.type = 1, this._$AH = A, this._$AN = void 0, this.element = t4, this.name = i4, this._$AM = e3, this.options = h3, s3.length > 2 || "" !== s3[0] || "" !== s3[1] ? (this._$AH = Array(s3.length - 1).fill(new String()), this.strings = s3) : this._$AH = A;
  }
  _$AI(t4, i4 = this, s3, e3) {
    const h3 = this.strings;
    let o2 = false;
    if (void 0 === h3) t4 = M(this, t4, i4, 0), o2 = !a(t4) || t4 !== this._$AH && t4 !== E, o2 && (this._$AH = t4);
    else {
      const e4 = t4;
      let n2, r2;
      for (t4 = h3[0], n2 = 0; n2 < h3.length - 1; n2++) r2 = M(this, e4[s3 + n2], i4, n2), r2 === E && (r2 = this._$AH[n2]), o2 ||= !a(r2) || r2 !== this._$AH[n2], r2 === A ? t4 = A : t4 !== A && (t4 += (r2 ?? "") + h3[n2 + 1]), this._$AH[n2] = r2;
    }
    o2 && !e3 && this.j(t4);
  }
  j(t4) {
    t4 === A ? this.element.removeAttribute(this.name) : this.element.setAttribute(this.name, t4 ?? "");
  }
};
var I = class extends H {
  constructor() {
    super(...arguments), this.type = 3;
  }
  j(t4) {
    this.element[this.name] = t4 === A ? void 0 : t4;
  }
};
var L = class extends H {
  constructor() {
    super(...arguments), this.type = 4;
  }
  j(t4) {
    this.element.toggleAttribute(this.name, !!t4 && t4 !== A);
  }
};
var z = class extends H {
  constructor(t4, i4, s3, e3, h3) {
    super(t4, i4, s3, e3, h3), this.type = 5;
  }
  _$AI(t4, i4 = this) {
    if ((t4 = M(this, t4, i4, 0) ?? A) === E) return;
    const s3 = this._$AH, e3 = t4 === A && s3 !== A || t4.capture !== s3.capture || t4.once !== s3.once || t4.passive !== s3.passive, h3 = t4 !== A && (s3 === A || e3);
    e3 && this.element.removeEventListener(this.name, this, s3), h3 && this.element.addEventListener(this.name, this, t4), this._$AH = t4;
  }
  handleEvent(t4) {
    "function" == typeof this._$AH ? this._$AH.call(this.options?.host ?? this.element, t4) : this._$AH.handleEvent(t4);
  }
};
var Z = class {
  constructor(t4, i4, s3) {
    this.element = t4, this.type = 6, this._$AN = void 0, this._$AM = i4, this.options = s3;
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AI(t4) {
    M(this, t4);
  }
};
var j = { M: h, P: o, A: n, C: 1, L: N, R, D: d, V: M, I: k, H, N: L, U: z, B: I, F: Z };
var B = t.litHtmlPolyfillSupport;
B?.(S, k), (t.litHtmlVersions ??= []).push("3.3.3");
var D = (t4, i4, s3) => {
  const e3 = s3?.renderBefore ?? i4;
  let h3 = e3._$litPart$;
  if (void 0 === h3) {
    const t5 = s3?.renderBefore ?? null;
    e3._$litPart$ = h3 = new k(i4.insertBefore(c(), t5), t5, void 0, s3 ?? {});
  }
  return h3._$AI(t4), h3;
};

// src/router.ts
var routes = [];
var notFoundRedirect = "/";
function route(pathPattern, handler) {
  const keys = [];
  const regexSource = pathPattern.replace(/:[a-zA-Z_]+/g, (match) => {
    keys.push(match.slice(1));
    return "([^/]+)";
  });
  routes.push({ pattern: new RegExp(`^${regexSource}$`), keys, handler });
}
function setNotFoundRedirect(path) {
  notFoundRedirect = path;
}
function currentPath() {
  const h3 = location.hash.replace(/^#/, "");
  return h3 || "/";
}
function dispatch() {
  const path = currentPath();
  for (const r2 of routes) {
    const m3 = r2.pattern.exec(path);
    if (m3) {
      const params = {};
      r2.keys.forEach((key, i4) => {
        params[key] = decodeURIComponent(m3[i4 + 1]);
      });
      r2.handler(params);
      return;
    }
  }
  location.hash = "#" + notFoundRedirect;
}
function startRouter() {
  window.addEventListener("hashchange", dispatch);
  dispatch();
}
function navigate(path) {
  location.hash = "#" + path;
}

// node_modules/lit-html/directive.js
var t2 = { ATTRIBUTE: 1, CHILD: 2, PROPERTY: 3, BOOLEAN_ATTRIBUTE: 4, EVENT: 5, ELEMENT: 6 };
var e2 = (t4) => (...e3) => ({ _$litDirective$: t4, values: e3 });
var i2 = class {
  constructor(t4) {
  }
  get _$AU() {
    return this._$AM._$AU;
  }
  _$AT(t4, e3, i4) {
    this._$Ct = t4, this._$AM = e3, this._$Ci = i4;
  }
  _$AS(t4, e3) {
    return this.update(t4, e3);
  }
  update(t4, e3) {
    return this.render(...e3);
  }
};

// node_modules/lit-html/directive-helpers.js
var { I: t3 } = j;
var i3 = (o2) => o2;
var s2 = () => document.createComment("");
var v2 = (o2, n2, e3) => {
  const l2 = o2._$AA.parentNode, d2 = void 0 === n2 ? o2._$AB : n2._$AA;
  if (void 0 === e3) {
    const i4 = l2.insertBefore(s2(), d2), n3 = l2.insertBefore(s2(), d2);
    e3 = new t3(i4, n3, o2, o2.options);
  } else {
    const t4 = e3._$AB.nextSibling, n3 = e3._$AM, c3 = n3 !== o2;
    if (c3) {
      let t5;
      e3._$AQ?.(o2), e3._$AM = o2, void 0 !== e3._$AP && (t5 = o2._$AU) !== n3._$AU && e3._$AP(t5);
    }
    if (t4 !== d2 || c3) {
      let o3 = e3._$AA;
      for (; o3 !== t4; ) {
        const t5 = i3(o3).nextSibling;
        i3(l2).insertBefore(o3, d2), o3 = t5;
      }
    }
  }
  return e3;
};
var u2 = (o2, t4, i4 = o2) => (o2._$AI(t4, i4), o2);
var m2 = {};
var p2 = (o2, t4 = m2) => o2._$AH = t4;
var M2 = (o2) => o2._$AH;
var h2 = (o2) => {
  o2._$AR(), o2._$AA.remove();
};

// node_modules/lit-html/directives/repeat.js
var u3 = (e3, s3, t4) => {
  const r2 = /* @__PURE__ */ new Map();
  for (let l2 = s3; l2 <= t4; l2++) r2.set(e3[l2], l2);
  return r2;
};
var c2 = e2(class extends i2 {
  constructor(e3) {
    if (super(e3), e3.type !== t2.CHILD) throw Error("repeat() can only be used in text expressions");
  }
  dt(e3, s3, t4) {
    let r2;
    void 0 === t4 ? t4 = s3 : void 0 !== s3 && (r2 = s3);
    const l2 = [], o2 = [];
    let i4 = 0;
    for (const s4 of e3) l2[i4] = r2 ? r2(s4, i4) : i4, o2[i4] = t4(s4, i4), i4++;
    return { values: o2, keys: l2 };
  }
  render(e3, s3, t4) {
    return this.dt(e3, s3, t4).values;
  }
  update(s3, [t4, r2, c3]) {
    const d2 = M2(s3), { values: p3, keys: a2 } = this.dt(t4, r2, c3);
    if (!Array.isArray(d2)) return this.ut = a2, p3;
    const h3 = this.ut ??= [], v3 = [];
    let m3, y2, x2 = 0, j2 = d2.length - 1, k2 = 0, w2 = p3.length - 1;
    for (; x2 <= j2 && k2 <= w2; ) if (null === d2[x2]) x2++;
    else if (null === d2[j2]) j2--;
    else if (h3[x2] === a2[k2]) v3[k2] = u2(d2[x2], p3[k2]), x2++, k2++;
    else if (h3[j2] === a2[w2]) v3[w2] = u2(d2[j2], p3[w2]), j2--, w2--;
    else if (h3[x2] === a2[w2]) v3[w2] = u2(d2[x2], p3[w2]), v2(s3, v3[w2 + 1], d2[x2]), x2++, w2--;
    else if (h3[j2] === a2[k2]) v3[k2] = u2(d2[j2], p3[k2]), v2(s3, d2[x2], d2[j2]), j2--, k2++;
    else if (void 0 === m3 && (m3 = u3(a2, k2, w2), y2 = u3(h3, x2, j2)), m3.has(h3[x2])) if (m3.has(h3[j2])) {
      const e3 = y2.get(a2[k2]), t5 = void 0 !== e3 ? d2[e3] : null;
      if (null === t5) {
        const e4 = v2(s3, d2[x2]);
        u2(e4, p3[k2]), v3[k2] = e4;
      } else v3[k2] = u2(t5, p3[k2]), v2(s3, d2[x2], t5), d2[e3] = null;
      k2++;
    } else h2(d2[j2]), j2--;
    else h2(d2[x2]), x2++;
    for (; k2 <= w2; ) {
      const e3 = v2(s3, v3[w2 + 1]);
      u2(e3, p3[k2]), v3[k2++] = e3;
    }
    for (; x2 <= j2; ) {
      const e3 = d2[x2++];
      null !== e3 && h2(e3);
    }
    return this.ut = a2, p2(s3, v3), E;
  }
});

// src/apiClient.ts
var BASE = "/api/v1";
var ApiError = class extends Error {
  status;
  constructor(status, message) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
};
var ApiNotFoundError = class extends ApiError {
  constructor(message) {
    super(404, message);
    this.name = "ApiNotFoundError";
  }
};
async function getJSON(path) {
  let res;
  try {
    res = await fetch(BASE + path);
  } catch (err) {
    throw new ApiError(0, `network error: ${err.message}`);
  }
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
    } catch {
    }
    if (res.status === 404) throw new ApiNotFoundError(message);
    throw new ApiError(res.status, message);
  }
  return await res.json();
}
function getInfo() {
  return getJSON("/info");
}
function listWorkflows(namespace = "") {
  const q = namespace ? `?namespace=${encodeURIComponent(namespace)}` : "";
  return getJSON(`/workflows${q}`);
}
function getWorkflow(id, version) {
  return getJSON(
    `/workflows/${encodeURIComponent(id)}/${encodeURIComponent(version)}`
  );
}
function listInstances(params = {}) {
  const sp = new URLSearchParams();
  if (params.namespace) sp.set("namespace", params.namespace);
  if (params.status) sp.set("status", params.status);
  if (params.definitionId) sp.set("definition_id", params.definitionId);
  if (params.limit) sp.set("limit", String(params.limit));
  if (params.offset) sp.set("offset", String(params.offset));
  const qs = sp.toString();
  return getJSON(`/instances${qs ? "?" + qs : ""}`);
}
function getInstance(id) {
  return getJSON(`/instances/${encodeURIComponent(id)}`);
}
function listEvents(id, params = {}) {
  const sp = new URLSearchParams();
  if (params.from) sp.set("from", String(params.from));
  if (params.limit) sp.set("limit", String(params.limit));
  const qs = sp.toString();
  return getJSON(
    `/instances/${encodeURIComponent(id)}/events${qs ? "?" + qs : ""}`
  );
}

// src/types.ts
var INSTANCE_STATUSES = [
  "pending",
  "running",
  "waiting",
  "completed",
  "failed",
  "cancelled",
  "compensating",
  "compensated",
  "compensation_failed"
];
var ACTIVE_STATUSES = /* @__PURE__ */ new Set([
  "pending",
  "running",
  "waiting",
  "compensating"
]);

// src/ui.ts
function loadingView(label = "Loading\u2026") {
  return b`<div class="state state-loading">${label}</div>`;
}
function emptyView(message) {
  return b`<div class="state state-empty">${message}</div>`;
}
function notFoundView(message) {
  return b`<div class="state state-notfound">${message}</div>`;
}
function errorView(err, onRetry) {
  const message = err instanceof Error ? err.message : String(err);
  return b`
    <div class="state state-error">
      <span>Could not load data: ${message}</span>
      <button @click=${onRetry}>Retry</button>
    </div>
  `;
}
function isNotFound(err) {
  return err instanceof ApiNotFoundError;
}
function errorMessage(err) {
  if (err instanceof ApiError) return err.message;
  return err instanceof Error ? err.message : String(err);
}
function relativeTimeFromMs(then) {
  if (Number.isNaN(then)) return "unknown";
  const diffMs = Date.now() - then;
  const diffS = Math.round(diffMs / 1e3);
  if (diffS < 5) return "just now";
  if (diffS < 60) return `${diffS}s ago`;
  const diffM = Math.round(diffS / 60);
  if (diffM < 60) return `${diffM}m ago`;
  const diffH = Math.round(diffM / 60);
  if (diffH < 24) return `${diffH}h ago`;
  const diffD = Math.round(diffH / 24);
  return `${diffD}d ago`;
}
function relativeTime(iso) {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return iso;
  return relativeTimeFromMs(then);
}
function freshnessView(fetchedAtMs) {
  return b`<span class="freshness" title="Last updated at ${new Date(fetchedAtMs).toLocaleTimeString()}"
    >as of ${relativeTimeFromMs(fetchedAtMs)}</span
  >`;
}
function statusBadge(status) {
  const cls = ACTIVE_STATUSES.has(status) ? "badge badge-active" : "badge badge-terminal";
  return b`<span class="${cls}">${status}</span>`;
}
function startVisibilityAwarePoll(fn, intervalMs) {
  let handle = null;
  function start() {
    if (handle !== null) return;
    handle = setInterval(fn, intervalMs);
  }
  function stop() {
    if (handle !== null) {
      clearInterval(handle);
      handle = null;
    }
  }
  function onVisibilityChange() {
    if (document.hidden) {
      stop();
    } else {
      fn();
      start();
    }
  }
  start();
  document.addEventListener("visibilitychange", onVisibilityChange);
  return () => {
    stop();
    document.removeEventListener("visibilitychange", onVisibilityChange);
  };
}

// src/onboarding.ts
function onboardingView(workflows) {
  const example = workflows[0];
  return b`
    <div class="onboarding">
      <h3>Welcome to AWIS</h3>
      <p>
        Nothing has run here yet — this GUI is read-only, so workflows are still created
        and started from the command line.
      </p>
      ${example ? b`<pre class="json-block">awis submit ${example.id}</pre>` : b`<pre class="json-block">awis init
awis submit hello-world</pre>`}
      <p class="onboarding-note">This page updates automatically once an instance exists.</p>
    </div>
  `;
}

// src/screens/instanceList.ts
var PAGE_SIZE = 50;
var POLL_MS = 15e3;
var TICK_MS = 1e3;
function mountInstanceList(container) {
  const state = {
    namespace: "",
    status: "",
    definitionId: "",
    offset: 0,
    loading: true,
    error: null,
    data: null,
    fetchedAt: 0,
    jumpId: "",
    onboardingWorkflows: null
  };
  let stopped = false;
  function draw() {
    D(view(state, actions), container);
  }
  async function load() {
    state.loading = state.data === null;
    state.error = null;
    draw();
    try {
      const resp = await listInstances({
        namespace: state.namespace || void 0,
        status: state.status || void 0,
        definitionId: state.definitionId || void 0,
        limit: PAGE_SIZE,
        offset: state.offset
      });
      if (stopped) return;
      const previousOrder = state.data ? state.data.instances.map((i4) => i4.instance_id) : [];
      const byId = new Map(resp.instances.map((i4) => [i4.instance_id, i4]));
      const previousSet = new Set(previousOrder);
      const sameSet = previousOrder.length === byId.size && previousOrder.every((id) => byId.has(id)) && [...byId.keys()].every((id) => previousSet.has(id));
      const ordered = sameSet && previousOrder.length > 0 ? previousOrder.map((id) => byId.get(id)) : [...resp.instances].sort(
        (a2, b2) => new Date(b2.updated_at).getTime() - new Date(a2.updated_at).getTime()
      );
      state.data = { instances: ordered, total: resp.total };
      state.fetchedAt = Date.now();
      const noFilters = !state.namespace && !state.status && !state.definitionId;
      if (resp.total === 0 && noFilters) {
        void checkOnboarding();
      } else {
        state.onboardingWorkflows = null;
      }
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }
  async function checkOnboarding() {
    try {
      const workflows = await listWorkflows();
      if (stopped) return;
      state.onboardingWorkflows = workflows;
    } catch {
      if (!stopped) state.onboardingWorkflows = null;
    }
    if (!stopped) draw();
  }
  const actions = {
    setNamespace(v3) {
      state.namespace = v3;
      state.offset = 0;
      void load();
    },
    setStatus(v3) {
      state.status = v3;
      state.offset = 0;
      void load();
    },
    setDefinitionId(v3) {
      state.definitionId = v3;
      state.offset = 0;
      void load();
    },
    setJumpId(v3) {
      state.jumpId = v3;
    },
    jumpToInstance() {
      const id = state.jumpId.trim();
      if (!id) return;
      navigate(`/instances/${encodeURIComponent(id)}`);
    },
    nextPage() {
      if (!state.data) return;
      if (state.offset + PAGE_SIZE >= state.data.total) return;
      state.offset += PAGE_SIZE;
      void load();
    },
    prevPage() {
      state.offset = Math.max(0, state.offset - PAGE_SIZE);
      void load();
    },
    retry() {
      void load();
    },
    openInstance(id) {
      navigate(`/instances/${encodeURIComponent(id)}`);
    }
  };
  void load();
  const stopPolling = startVisibilityAwarePoll(() => void load(), POLL_MS);
  const tickHandle = setInterval(() => {
    if (state.fetchedAt > 0) draw();
  }, TICK_MS);
  return () => {
    stopped = true;
    stopPolling();
    clearInterval(tickHandle);
  };
}
function view(state, actions) {
  return b`
    <section class="screen">
      <div class="screen-header">
        <h2>Instances</h2>
        ${state.fetchedAt > 0 ? freshnessView(state.fetchedAt) : ""}
      </div>
      <div class="filters">
        <label
          >Namespace
          <input
            type="text"
            placeholder="All namespaces"
            .value=${state.namespace}
            @change=${(e3) => actions.setNamespace(e3.target.value)}
          />
        </label>
        <label
          >Status
          <select @change=${(e3) => actions.setStatus(e3.target.value)}>
            <option value="" ?selected=${state.status === ""}>All statuses</option>
            ${INSTANCE_STATUSES.map(
    (s3) => b`<option value=${s3} ?selected=${state.status === s3}>${s3}</option>`
  )}
          </select>
        </label>
        <label
          >Workflow
          <input
            type="text"
            placeholder="All workflows"
            .value=${state.definitionId}
            @change=${(e3) => actions.setDefinitionId(e3.target.value)}
          />
        </label>
        <label
          >Jump to instance
          <form
            @submit=${(e3) => {
    e3.preventDefault();
    actions.jumpToInstance();
  }}
          >
            <input
              type="text"
              placeholder="Paste an instance ID"
              .value=${state.jumpId}
              @input=${(e3) => actions.setJumpId(e3.target.value)}
            />
          </form>
        </label>
      </div>
      ${bodyView(state, actions)}
    </section>
  `;
}
function bodyView(state, actions) {
  if (state.loading) return loadingView("Loading instances\u2026");
  if (state.error) {
    return b`${errorView(state.error, actions.retry)}${state.data ? dataView(state.data, state.offset, actions) : ""}`;
  }
  if (!state.data || state.data.instances.length === 0) {
    if (state.onboardingWorkflows !== null) {
      return onboardingView(state.onboardingWorkflows);
    }
    return emptyView("No instances match these filters.");
  }
  return dataView(state.data, state.offset, actions);
}
function dataView(data, offset, actions) {
  const { instances, total } = data;
  const page = Math.floor(offset / PAGE_SIZE) + 1;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  return b`
    <table class="data-table">
      <thead>
        <tr>
          <th>Instance</th>
          <th>Definition</th>
          <th>Version</th>
          <th>Status</th>
          <th>Current Step(s)</th>
          <th>Updated</th>
        </tr>
      </thead>
      <tbody>
        ${c2(
    instances,
    (inst) => inst.instance_id,
    (inst) => b`
            <tr class="clickable-row" @click=${() => actions.openInstance(inst.instance_id)}>
              <td class="mono" title=${inst.instance_id}>${inst.instance_id.slice(0, 8)}…</td>
              <td>${inst.definition_id}</td>
              <td>${inst.version}</td>
              <td>${statusBadge(inst.status)}</td>
              <td>${inst.current_steps.join(", ") || "\u2014"}</td>
              <td title=${inst.updated_at}>${relativeTime(inst.updated_at)}</td>
            </tr>
          `
  )}
      </tbody>
    </table>
    <div class="pagination">
      <button ?disabled=${offset === 0} @click=${actions.prevPage}>Prev</button>
      <span>Page ${page} of ${pageCount} (${total} total)</span>
      <button ?disabled=${offset + PAGE_SIZE >= total} @click=${actions.nextPage}>Next</button>
    </div>
  `;
}

// src/screens/instanceDetail.ts
var POLL_MS2 = 15e3;
var TICK_MS2 = 1e3;
function mountInstanceDetail(container, instanceId) {
  const state = { loading: true, error: null, data: null, fetchedAt: 0 };
  let stopped = false;
  function draw() {
    D(view2(state, actions, instanceId), container);
  }
  async function load() {
    state.loading = state.data === null;
    state.error = null;
    draw();
    try {
      const data = await getInstance(instanceId);
      if (stopped) return;
      state.data = data;
      state.fetchedAt = Date.now();
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }
  const actions = {
    retry() {
      void load();
    },
    viewEvents() {
      navigate(`/instances/${encodeURIComponent(instanceId)}/events`);
    }
  };
  void load();
  const stopPolling = startVisibilityAwarePoll(() => void load(), POLL_MS2);
  const tickHandle = setInterval(() => {
    if (state.data) draw();
  }, TICK_MS2);
  return () => {
    stopped = true;
    stopPolling();
    clearInterval(tickHandle);
  };
}
function liveCountdown(state) {
  if (!state.data || state.data.timeout_remaining_s === null) return null;
  const elapsedS = Math.floor((Date.now() - state.fetchedAt) / 1e3);
  return Math.max(0, state.data.timeout_remaining_s - elapsedS);
}
function formatCountdown(seconds) {
  const m3 = Math.floor(seconds / 60);
  const s3 = seconds % 60;
  return `${m3}m ${s3}s remaining`;
}
function view2(state, actions, instanceId) {
  return b`
    <section class="screen">
      <div class="screen-header">
        <h2>Instance ${instanceId}</h2>
        ${state.fetchedAt > 0 ? freshnessView(state.fetchedAt) : ""}
      </div>
      ${bodyView2(state, actions)}
    </section>
  `;
}
function bodyView2(state, actions) {
  if (state.error && isNotFound(state.error)) {
    return notFoundView(`This instance does not exist. (${errorMessage(state.error)})`);
  }
  if (state.loading && !state.data) return loadingView("Loading instance\u2026");
  if (state.error) {
    return b`${errorView(state.error, actions.retry)}${state.data ? dataView2(state.data, liveCountdown(state), actions) : ""}`;
  }
  if (!state.data) return loadingView("Loading instance\u2026");
  return dataView2(state.data, liveCountdown(state), actions);
}
function dataView2(d2, countdown, actions) {
  return b`
    <dl class="detail-grid">
      <dt>Status</dt>
      <dd>${statusBadge(d2.status)}</dd>
      <dt>Definition</dt>
      <dd>${d2.definition_id} @ ${d2.version}</dd>
      <dt>Current Step(s)</dt>
      <dd>${d2.current_steps.join(", ") || "\u2014"}</dd>
      <dt>Created</dt>
      <dd>${d2.created_at}</dd>
      <dt>Updated</dt>
      <dd>${d2.updated_at}</dd>
    </dl>

    ${d2.status === "waiting" ? b`
          <div class="wait-info">
            <h3>Waiting</h3>
            <p>Signal: <strong>${d2.signal_name ?? "\u2014"}</strong></p>
            <p>${countdown === null ? "No timeout configured" : formatCountdown(countdown)}</p>
          </div>
        ` : ""}

    <details>
      <summary>Inputs</summary>
      <pre class="json-block">${JSON.stringify(d2.inputs ?? {}, null, 2)}</pre>
    </details>
    <details>
      <summary>Outputs</summary>
      <pre class="json-block">${JSON.stringify(d2.outputs ?? {}, null, 2)}</pre>
    </details>

    <button class="primary" @click=${actions.viewEvents}>View Events</button>
  `;
}

// src/eventFormatters.ts
function str(v3, fallback = "?") {
  return typeof v3 === "string" || typeof v3 === "number" ? String(v3) : fallback;
}
var FORMATTERS = {
  WorkflowStarted: () => "Workflow started",
  StepStarted: (p3, stepId) => `Step ${stepId} started (attempt ${str(p3.attempt)})`,
  StepCompleted: (p3, stepId) => `Step ${stepId} completed in ${str(p3.duration_ms)}ms`,
  StepFailed: (p3, stepId) => `Step ${stepId} failed: ${str(p3.error, "unknown error")}` + (p3.retrying ? " (retrying)" : ""),
  StepFallbackActivated: (p3, stepId) => `Step ${stepId} fell back to ${str(p3.fallback_step_id)}: ${str(p3.reason, "")}`,
  SignalReceived: (p3) => `Signal '${str(p3.signal_name)}' received`,
  WorkflowCompleted: (p3) => `Workflow completed in ${str(p3.duration_ms)}ms`,
  WorkflowFailed: (p3, stepId) => `Workflow failed at ${stepId || str(p3.step_id)}: ${str(p3.error, "")}`,
  WorkflowCancelled: (p3) => `Workflow cancelled: ${str(p3.reason, "")}`,
  WorkflowCompensating: (p3) => `Compensation started from ${str(p3.from_step)}`,
  WorkflowCompensated: () => "Compensation completed",
  WorkflowCompensationFailed: (p3, stepId) => `Compensation failed at ${stepId || str(p3.step_id)}: ${str(p3.error, "")}`
};
function summarizeEvent(eventType, stepId, payload) {
  const fn = FORMATTERS[eventType];
  if (!fn) return eventType;
  try {
    return fn(payload ?? {}, stepId);
  } catch {
    return eventType;
  }
}

// src/screens/eventTimeline.ts
var PAGE_SIZE2 = 200;
function mountEventTimeline(container, instanceId) {
  const state = {
    loading: true,
    loadingMore: false,
    error: null,
    events: [],
    nextCursor: null,
    expandedIds: /* @__PURE__ */ new Set()
  };
  let stopped = false;
  function draw() {
    D(view3(state, actions, instanceId), container);
  }
  async function loadFirstPage() {
    state.loading = true;
    state.error = null;
    state.events = [];
    state.nextCursor = null;
    draw();
    try {
      const resp = await listEvents(instanceId, { limit: PAGE_SIZE2 });
      if (stopped) return;
      state.events = resp.events ?? [];
      state.nextCursor = resp.next_cursor ?? null;
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }
  async function loadMore() {
    if (state.nextCursor === null || state.loadingMore) return;
    state.loadingMore = true;
    draw();
    try {
      const resp = await listEvents(instanceId, { from: state.nextCursor, limit: PAGE_SIZE2 });
      if (stopped) return;
      state.events = state.events.concat(resp.events ?? []);
      state.nextCursor = resp.next_cursor ?? null;
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loadingMore = false;
        draw();
      }
    }
  }
  const actions = {
    retry() {
      void loadFirstPage();
    },
    loadMore() {
      void loadMore();
    },
    backToInstance() {
      navigate(`/instances/${encodeURIComponent(instanceId)}`);
    },
    toggleExpanded(eventId) {
      if (state.expandedIds.has(eventId)) {
        state.expandedIds.delete(eventId);
      } else {
        state.expandedIds.add(eventId);
      }
      draw();
    }
  };
  void loadFirstPage();
  return () => {
    stopped = true;
  };
}
function view3(state, actions, instanceId) {
  return b`
    <section class="screen">
      <button class="link-button" @click=${actions.backToInstance}>&larr; Back to instance</button>
      <h2>Events for ${instanceId}</h2>
      ${bodyView3(state, actions)}
    </section>
  `;
}
function bodyView3(state, actions) {
  if (state.error && isNotFound(state.error)) {
    return notFoundView(`This instance does not exist. (${errorMessage(state.error)})`);
  }
  if (state.loading) return loadingView("Loading events\u2026");
  if (state.error) {
    return b`${errorView(state.error, actions.retry)}${state.events.length > 0 ? eventsView(state, actions) : ""}`;
  }
  if (state.events.length === 0) return emptyView("No events yet.");
  return eventsView(state, actions);
}
function eventsView(state, actions) {
  return b`
    <ol class="event-list">
      ${state.events.map((ev) => eventRow(ev, state.expandedIds.has(ev.event_id), actions))}
    </ol>
    ${state.nextCursor !== null ? b`<button ?disabled=${state.loadingMore} @click=${actions.loadMore}>
          ${state.loadingMore ? "Loading\u2026" : "Load more"}
        </button>` : ""}
  `;
}
function eventRow(ev, expanded, actions) {
  return b`
    <li class="event-row">
      <span class="event-time" title=${ev.emitted_at}>${ev.emitted_at}</span>
      <span class="event-type">${ev.event_type}</span>
      <span class="event-summary">${summarizeEvent(ev.event_type, ev.step_id, ev.payload)}</span>
      <details ?open=${expanded} @toggle=${() => actions.toggleExpanded(ev.event_id)}>
        <summary>raw</summary>
        ${expanded ? b`<pre class="json-block">${JSON.stringify(ev.payload, null, 2)}</pre>` : ""}
      </details>
    </li>
  `;
}

// src/screens/workflowList.ts
function mountWorkflowList(container) {
  const state = { namespace: "", loading: true, error: null, data: null };
  let stopped = false;
  function draw() {
    D(view4(state, actions), container);
  }
  async function load() {
    state.loading = state.data === null;
    state.error = null;
    draw();
    try {
      const data = await listWorkflows(state.namespace || void 0);
      if (stopped) return;
      state.data = data;
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }
  const actions = {
    setNamespace(v3) {
      state.namespace = v3;
      void load();
    },
    retry() {
      void load();
    },
    openWorkflow(id, version) {
      navigate(`/workflows/${encodeURIComponent(id)}/${encodeURIComponent(version)}`);
    }
  };
  void load();
  return () => {
    stopped = true;
  };
}
function view4(state, actions) {
  return b`
    <section class="screen">
      <h2>Workflows</h2>
      <div class="filters">
        <label
          >Namespace
          <input
            type="text"
            placeholder="All namespaces"
            .value=${state.namespace}
            @change=${(e3) => actions.setNamespace(e3.target.value)}
          />
        </label>
      </div>
      ${bodyView4(state, actions)}
    </section>
  `;
}
function bodyView4(state, actions) {
  if (state.loading) return loadingView("Loading workflows\u2026");
  if (state.error) {
    return b`${errorView(state.error, actions.retry)}${state.data && state.data.length > 0 ? dataView3(state.data, actions) : ""}`;
  }
  if (!state.data || state.data.length === 0) {
    return emptyView("No workflows registered in this namespace.");
  }
  return dataView3(state.data, actions);
}
function dataView3(data, actions) {
  return b`
    <table class="data-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Version</th>
          <th>Namespace</th>
          <th>Steps</th>
        </tr>
      </thead>
      <tbody>
        ${data.map(
    (w2) => b`
            <tr class="clickable-row" @click=${() => actions.openWorkflow(w2.id, w2.version)}>
              <td>${w2.id}</td>
              <td>${w2.version}</td>
              <td>${w2.namespace}</td>
              <td>${w2.step_count}</td>
            </tr>
          `
  )}
      </tbody>
    </table>
  `;
}

// src/screens/workflowDetail.ts
function mountWorkflowDetail(container, id, version) {
  const state = { loading: true, error: null, data: null };
  let stopped = false;
  function draw() {
    D(view5(state, actions, id, version), container);
  }
  async function load() {
    state.loading = true;
    state.error = null;
    draw();
    try {
      const data = await getWorkflow(id, version);
      if (stopped) return;
      state.data = data;
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }
  const actions = {
    retry() {
      void load();
    }
  };
  void load();
  return () => {
    stopped = true;
  };
}
function view5(state, actions, id, version) {
  return b`
    <section class="screen">
      <h2>Workflow ${id} @ ${version}</h2>
      ${bodyView5(state, actions)}
    </section>
  `;
}
function bodyView5(state, actions) {
  if (state.error) {
    return isNotFound(state.error) ? notFoundView(`This workflow does not exist. (${errorMessage(state.error)})`) : errorView(state.error, actions.retry);
  }
  if (state.loading || !state.data) return loadingView("Loading workflow\u2026");
  return dataView4(state.data);
}
function dataView4(def) {
  return b`
    <dl class="detail-grid">
      <dt>Name</dt>
      <dd>${def.name}</dd>
      <dt>Namespace</dt>
      <dd>${def.namespace}</dd>
      ${def.description ? b`<dt>Description</dt>
            <dd>${def.description}</dd>` : ""}
    </dl>

    <h3>Steps</h3>
    <table class="data-table">
      <thead>
        <tr>
          <th>Step ID</th>
          <th>Name</th>
          <th>Type</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        ${def.steps.map((s3) => stepRow(s3, def))}
      </tbody>
    </table>

    <h3>Transitions</h3>
    ${def.transitions.length === 0 ? b`<p class="state state-empty">No transitions (single-step workflow).</p>` : b`
          <table class="data-table">
            <thead>
              <tr>
                <th>From</th>
                <th>To</th>
                <th>Condition</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              ${def.transitions.map((t4) => transitionRow(t4))}
            </tbody>
          </table>
        `}

    <h3>Triggers</h3>
    <ul class="trigger-list">
      ${def.triggers.map(
    (t4) => b`
          <li>
            <strong>${t4.type}</strong>
            ${Object.keys(t4.config).length > 0 ? b`<pre class="json-block">${JSON.stringify(t4.config, null, 2)}</pre>` : ""}
          </li>
        `
  )}
    </ul>
  `;
}
function stepRow(s3, def) {
  const badges = [];
  if (s3.id === def.initial_step) badges.push(b`<span class="badge badge-active">initial</span>`);
  if (def.final_steps.includes(s3.id)) badges.push(b`<span class="badge badge-terminal">final</span>`);
  return b`
    <tr>
      <td class="mono">${s3.id} ${badges}</td>
      <td>${s3.name}</td>
      <td><span class="badge">${s3.type}</span></td>
      <td>${stepDetails(s3)}</td>
    </tr>
  `;
}
function stepDetails(s3) {
  return b`
    <div class="step-details">
      ${typeSpecificDetails(s3)}
      ${s3.fallback ? b`<div>fallback: <span class="mono">${s3.fallback}</span></div>` : ""}
    </div>
  `;
}
function typeSpecificDetails(s3) {
  if (s3.type === "intelligence" && s3.intelligence) {
    const iq = s3.intelligence;
    return b`
      <div>capability: <strong>${iq.capability}</strong></div>
      ${iq.model_hint ? b`<div>model_hint: ${iq.model_hint}</div>` : ""}
      ${iq.context_budget ? b`<div>context_budget: ${iq.context_budget}</div>` : ""}
      ${iq.required ? b`<div>required: true</div>` : ""}
    `;
  }
  if (s3.type === "signal" && s3.wait_signal) {
    const ws = s3.wait_signal;
    return b`
      <div>signal: <strong>${ws.signal_name}</strong></div>
      ${ws.timeout ? b`<div>timeout: ${ws.timeout}</div>` : ""}
      <div>timeout_action: ${ws.timeout_action}</div>
    `;
  }
  return b`
    ${s3.handler ? b`<div>handler: <span class="mono">${s3.handler}</span></div>` : ""}
    ${s3.retry ? b`<div>retry: ${s3.retry.attempts}× (${s3.retry.backoff})</div>` : ""}
    ${s3.timeout ? b`<div>timeout: ${s3.timeout}</div>` : ""}
  `;
}
function transitionRow(t4) {
  return b`
    <tr>
      <td class="mono">${t4.from}</td>
      <td class="mono">${t4.to}</td>
      <td>${t4.condition || "\u2014"}</td>
      <td>${t4.on_error ? b`<span class="badge badge-terminal">on error</span>` : ""}</td>
    </tr>
  `;
}

// src/main.ts
var appEl = document.getElementById("app");
if (!appEl) throw new Error("main.ts: #app container not found in index.html");
var cleanupCurrentScreen = null;
function mount(fn) {
  if (cleanupCurrentScreen) {
    cleanupCurrentScreen();
    cleanupCurrentScreen = null;
  }
  appEl.replaceChildren();
  const screenContainer = document.createElement("div");
  appEl.appendChild(screenContainer);
  cleanupCurrentScreen = fn(screenContainer);
}
route("/instances", () => mount((c3) => mountInstanceList(c3)));
route("/instances/:id", (params) => mount((c3) => mountInstanceDetail(c3, params.id)));
route("/instances/:id/events", (params) => mount((c3) => mountEventTimeline(c3, params.id)));
route("/workflows", () => mount((c3) => mountWorkflowList(c3)));
route("/workflows/:id/:version", (params) => mount((c3) => mountWorkflowDetail(c3, params.id, params.version)));
setNotFoundRedirect("/instances");
function setupNav() {
  const links = document.querySelectorAll("[data-nav]");
  links.forEach((a2) => {
    a2.addEventListener("click", (e3) => {
      e3.preventDefault();
      navigate(a2.getAttribute("data-nav"));
    });
  });
}
var CONN_POLL_MS = 1e4;
function startConnectionIndicator() {
  const el = document.getElementById("conn-indicator");
  if (!el) return;
  let status = "unknown";
  let version = "";
  function draw() {
    const cls = status === "ok" ? "conn-dot conn-ok" : status === "down" ? "conn-dot conn-down" : "conn-dot";
    D(
      b`<span class="conn-indicator">
        <span class="${cls}"></span>
        ${version ? b`v${version}` : ""}
      </span>`,
      el
    );
  }
  async function check() {
    try {
      const info = await getInfo();
      status = "ok";
      version = info.version;
    } catch {
      status = "down";
    }
    draw();
  }
  void check();
  startVisibilityAwarePoll(() => void check(), CONN_POLL_MS);
}
setupNav();
startConnectionIndicator();
startRouter();
/*! Bundled license information:

lit-html/lit-html.js:
lit-html/directive.js:
lit-html/directives/repeat.js:
  (**
   * @license
   * Copyright 2017 Google LLC
   * SPDX-License-Identifier: BSD-3-Clause
   *)

lit-html/directive-helpers.js:
  (**
   * @license
   * Copyright 2020 Google LLC
   * SPDX-License-Identifier: BSD-3-Clause
   *)
*/
//# sourceMappingURL=bundle.js.map
