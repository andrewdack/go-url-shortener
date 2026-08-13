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

## CORS

The Go API now enables CORS for the local Vite origins
(`http://localhost:5173` and `http://127.0.0.1:5173`) and the GitHub Pages origin
(`https://andrewdack.github.io`). It allows the frontend's JSON `GET`, `POST`, `PATCH`, and
`DELETE` requests, including browser preflight `OPTIONS` requests.

For a deployed frontend, set `VITE_API_BASE_URL` to the public backend origin and ensure the
backend's allowed-origin list includes the exact frontend origin. The origin should not include
the `/go-url-shortener/` path.
