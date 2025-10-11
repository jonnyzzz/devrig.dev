# devrig.dev Website

The official website for devrig, built with Hugo, TypeScript, React, and Webpack.

## Development

Run the development server:

```bash
./dev.sh
```

This starts both Node.js (for TypeScript/React) and Hugo in watch mode.
The site will be available at http://localhost:1313

## Production Build

Build the production website:

```bash
./build.sh
```

The production files will be generated in `./public/`

## Build Process

The build uses Docker containers with pinned versions:

1. **Node.js** (20.18.1-alpine3.20): Install dependencies and compile TypeScript/React with Webpack
2. **Hugo** (0.141.0): Generate static site with minification
3. Auto-generate `download.md` from `latest.json`

## Dependencies

All dependencies use fixed versions:
- React: 18.3.1
- TypeScript: 5.7.3
- Webpack: 5.97.1
- Hugo: 0.141.0
- Node.js: 20.18.1

## Structure

- `content/` - Markdown content files
- `themes/devrig-minimal/` - Custom Hugo theme
- `static/` - Static assets (CSS, downloads)
- `src/` - TypeScript/React source files
- `scripts/` - Build scripts (Node.js)
- `docker-compose.dev.yml` - Development environment
- `public/` - Generated production site (git-ignored)

## Updating Downloads

Edit `static/download/latest.json` to update release information. The `download.md` page will be automatically generated during the build.
