# Tiny Route frontend

Vue 3 + Vite frontend for the Go URL shortener. It supports:

- Creating and copying short links
- Keeping a browser-local list of links
- Viewing click counts
- Updating destinations
- Deleting links

There is no login system. The frontend creates an anonymous browser ID and stores it with the
link list in `localStorage`. Clearing browser storage removes the local management history.

## Local development

Start the Redis-backed API from the repository root:

```sh
docker compose up --build
```

Then start Vite in this directory:

```sh
npm install
npm run dev
```

The development API origin is configured in `.env.development` and defaults to
`http://localhost:9808` in the application code.

## Production

Set `VITE_API_BASE_URL` in `.env.production` (or the deployment environment) to the public API
origin before building:

```sh
npm run build
```

The Vite base path must remain `/go-url-shortener/` for GitHub Pages.

## CORS prerequisite

The Go backend does not currently enable CORS. Browser API requests from the Vite dev server or
GitHub Pages will be blocked until the backend allows the appropriate frontend origin and request
methods (`GET`, `POST`, `PATCH`, `DELETE`, and `OPTIONS`).
