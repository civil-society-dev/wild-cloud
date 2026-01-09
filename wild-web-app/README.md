# Wild Cloud Web App

The Wild Cloud Web App is a web-based interface for managing Wild Cloud instances. It allows users to view and control their Wild Cloud environments, including deploying applications, monitoring resources, and configuring settings.

## Prerequisites

Before starting the web app, ensure the Wild Central API is running:

```bash
cd ../api
make dev
```

The API should be accessible at `http://localhost:5055`.

## Development

### Initial Setup

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Update `.env` if your API is running on a different host/port:
```bash
VITE_API_BASE_URL=http://localhost:5055
```

3. Install dependencies:
```bash
pnpm install
```

4. Start the development server:
```bash
pnpm run dev
```

The web app will be available at `http://localhost:5173` (or the next available port).

## Other Scripts

```bash
pnpm run build # Build the project
pnpm run lint # Lint the codebase
pnpm run preview # Preview the production build
pnpm run type-check # Type check the codebase
pnpm run test # Run tests
pnpm run test:ui # Run tests with UI
pnpm run test:coverage # Run tests with coverage report
pnpm run build:css # Build the CSS using Tailwind
pnpm run check # Run lint, type-check, and tests
```

## Environment Variables

### `VITE_API_BASE_URL`

The base URL of the Wild Central API server.

- **Default:** `http://localhost:5055`
- **Example:** `http://192.168.1.100:5055`
- **Usage:** Set in `.env` file (see `.env.example` for template)

This variable is used by the API client to connect to the Wild Central API. If not set, it defaults to `http://localhost:5055`.

