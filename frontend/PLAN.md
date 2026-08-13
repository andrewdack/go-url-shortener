# Frontend build plan — go-url-shortener

Self-contained brief for building the frontend UI. You (the agent picking this up) have not seen prior conversation about this project — everything you need is below. Do not assume anything about the backend beyond what's documented here; verify against the actual Go source in `../handler/handlers.go` and `../router/router.go` if anything looks off.

## What exists already

- `frontend/` is a Vite + Vue 3 project, scaffolded via `npm create vite@latest frontend -- --template vue`, dependencies installed, builds clean (`npm run build`).
- `frontend/vite.config.js` has `base: '/go-url-shortener/'` set — required because this deploys to GitHub Pages as a project site (`https://andrewdack.github.io/go-url-shortener/`), not a root domain. Do not remove this.
- `.github/workflows/deploy-frontend.yml` already builds and deploys `frontend/dist` to GitHub Pages on every push to `main` that touches `frontend/**`. Nothing to change here unless the build output path changes.
- The backend is a Go/Gin API in the repo root (`main.go`, `handler/`, `router/`, `store/`), backed by Redis, containerized (`Dockerfile`, `compose.yaml`). It is NOT yet deployed anywhere public — assume `http://localhost:9808` for local dev against `docker compose up`.

## Design direction

The user explicitly likes the current Vite+Vue scaffold's default aesthetic and wants it preserved/extended, not replaced. Specifically: the bordered-container layout with horizontal divider lines between sections, and the small "tick" marks at the ends of those dividers. Look at `frontend/src/style.css` — the relevant pieces are:

- `#app { border-inline: 1px solid var(--border); }` — a full-height container with vertical border lines on both sides.
- Horizontal `border-top`/`border-bottom` dividers separating sections (see `#next-steps`, `#spacer`).
- `.ticks` — a CSS class that draws small triangular tick marks at the left/right ends of a divider line (pure CSS, `::before`/`::after` with `border` triangles). Reuse this for any new section dividers you add.
- CSS custom properties (`--text`, `--accent`, `--border`, `--bg`, etc.) already define a light/dark-mode-aware palette via `@media (prefers-color-scheme: dark)`. Use these variables for any new UI, don't hardcode colors — dark mode support already works, don't break it.

Build new views/components as extensions of this system: bordered sections stacked vertically, divided by ticked lines, consistent typography via the existing `h1`/`h2`/`code` styles. Do not introduce a component library (no Vuetify/PrimeVue/etc.) or a different design language — this is meant to look like it belongs in the same scaffold, not a redesign.

## Backend API contract

Base URL should be configurable, not hardcoded — read it from `import.meta.env.VITE_API_BASE_URL`, falling back to `http://localhost:9808` for local dev. Set up a `.env.development` with that fallback and leave `.env.production` for whoever deploys the backend to fill in with the real host later (do not invent a production URL).

All endpoints, verified against current handler source:

### `POST /create-short-url`
Rate limited: 10 requests/minute per client IP (returns `429` past that — handle this in the UI, don't just show a generic error).

Request body:
```json
{ "longUrl": "https://example.com/page", "userId": "some-string" }
```
Both fields required. `longUrl` must pass Go's URL validation (needs a scheme, e.g. `https://` — bare `example.com` is rejected with `400`).

Success (`200`):
```json
{ "message": "Short URL created successfully", "shortUrl": "http://localhost:9808/abc12345" }
```
Note `shortUrl` is the full URL including host, ready to display/copy as-is.

Important behavior: short codes are deterministic — hashing `longUrl + userId`. The same user submitting the same URL twice gets back the *same* short code both times (not an error, just idempotent). Don't design the UI to treat a repeat submission as a failure case.

Errors: `400` (bad/missing body), `429` (rate limited), `500` (backend error) — all return `{ "error": "..." }`.

### `GET /:shortUrl` (redirect)
Not something the frontend calls via fetch — this is what the short link itself points to (browser navigates there directly, e.g. `<a href>`). `302` redirect to the original URL on success, `404` if the code doesn't exist/expired. Only relevant to the frontend as "this is what the generated link does when clicked" — no UI needed for it beyond linking to it.

### `GET /:shortUrl/count`
Success (`200`):
```json
{ "shortUrl": "abc12345", "clicks": 3 }
```
`404` if the short URL doesn't exist. No ownership check — anyone can view any link's click count (backend doesn't restrict this).

### `PATCH /:shortUrl` (update destination)
Request body:
```json
{ "longUrl": "https://example.com/new-destination", "userId": "some-string" }
```
`userId` must match the `userId` used at creation time (ownership check).

Success (`200`):
```json
{ "message": "Short URL updated successfully", "shortUrl": "abc12345", "longUrl": "https://example.com/new-destination" }
```
Errors: `400` (bad body), `403` (`userId` doesn't match the owner — `{"error": "user does not own short url"}`), `404` (short URL doesn't exist), `500`.

### `DELETE /:shortUrl`
Request body:
```json
{ "userId": "some-string" }
```
Same ownership check as PATCH. Success (`200`): `{ "message": "Short URL deleted successfully", "shortUrl": "abc12345" }`. Same error shapes as PATCH (`403`/`404`/`500`).

## Critical design decision: there is no real auth

`userId` is **not** an authenticated identity — it's a free-text string the backend uses purely as an ownership token (whoever supplies the matching string can update/delete a link). There is no login system, no password, nothing to build on the backend side for this.

Handle it like this: on first visit, generate a random ID (`crypto.randomUUID()`) and persist it in `localStorage`. Use that automatically as `userId` for every create/update/delete call this browser makes — the user never sees or types it. This means:
- Links created in one browser can't be managed from another browser/device (no real accounts) — that's expected, not a bug to fix.
- Don't build a login/signup UI. There's nothing on the backend for it to talk to.

## Suggested pages/views (vue-router)

You'll need routing since there's more than one logical view. Install `vue-router` (not present yet).

1. **Home / Create** (`/`) — form with a single `longUrl` input, submit calls `POST /create-short-url` (with the localStorage `userId` auto-attached, invisible to the user). On success, show the generated short link with a copy-to-clipboard button. Handle `429` distinctly ("you're doing that too fast, wait a bit") vs `400` (show the validation error) vs `500`.
2. **My Links** (`/links`) — since there's no backend endpoint to list "all links for a userId" (check: there genuinely isn't one — grep `router/router.go` to confirm before assuming), this has to be client-side: keep a local list in `localStorage` of `{shortCode, longUrl, createdAt}` every time this browser successfully creates a link, and render that list here. For each entry, fetch `GET /:shortCode/count` to show click counts, and provide inline edit (PATCH) / delete (DELETE) actions.
3. Optionally, a per-link detail view (`/links/:shortCode`) if the "My Links" list gets busy — not required, use judgement.

## Explicit non-goals

- Don't build real authentication.
- Don't try to add a "list all short URLs" backend endpoint — it doesn't exist and isn't in scope for this task; work within `localStorage` as described above. If you think a backend change is genuinely required, stop and flag it rather than modifying Go files unprompted.
- Don't change the deploy workflow or `vite.config.js` base path.
- Don't introduce a component library or state management library (Pinia, etc.) — this app is small enough for plain `ref`/`reactive` and route params; don't over-engineer it.

## One backend prerequisite (not yet done, blocks real end-to-end testing against the deployed API)

The Go backend has **no CORS configuration** — grep `main.go`/`router/router.go` for "cors" to confirm, currently nothing. Once the frontend is deployed to `github.io` and the backend to wherever it ends up (Railway/Render, TBD), cross-origin `fetch` calls will be blocked by the browser until `gin-contrib/cors` (or equivalent) is added to allow the deployed frontend's origin. This is a backend change, small, but out of scope for a frontend-focused task unless you're asked to do it too — flag it rather than silently skipping around it (e.g. don't build something that only works because of `--disable-web-security` or a browser extension).

## Verifying your work

- `cd frontend && npm run dev` — Vite dev server, point at a locally running backend (`docker compose up` from repo root, port `9808`).
- `npm run build` — must succeed with no errors before considering any milestone done.
- Test against a live backend, not mocks — create a link, click it (confirm redirect + click count increments), update it, delete it, and confirm a 429 actually triggers if you spam create.
