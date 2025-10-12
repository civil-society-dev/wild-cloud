# Building Wild App

This document describes the architecture and tooling used to build the Wild App, the web-based interface for managing Wild Cloud instances, hosted on Wild Central.

## Principles

- Stick with well known standards.
- Keep it simple.
- Use popular, well-maintained libraries.
- Use components wherever possible to avoid reinventing the wheel.
- Use TypeScript for type safety.

### Tooling
## Dev Environment Requirements

- Node.js 20+
- pnpm for package management
- vite for build tooling
- React + TypeScript
- Tailwind CSS for styling
- shadcn/ui for ready-made components
- radix-ui for accessible components
- eslint for linting
- tsc for type checking
- vitest for unit tests

#### Makefile commands

- Build application: `make app-build`
- Run locally: `make app-run`
- Format code: `make app-fmt`
- Lint and typecheck: `make app-check`
- Test installation: `make app-test`

### Scaffolding apps

It is important to start an app with a good base structure to avoid difficult to debug config issues.

This is a recommended setup.

#### Install pnpm if necessary:

```bash
curl -fsSL https://get.pnpm.io/install.sh | sh -
```

#### Install a React + Speedy Web Compiler (SWC) + TypeScript + TailwindCSS app with vite:

```
pnpm create vite@latest my-app -- --template react-swc-ts
```

#### Reconfigure to use shadcn/ui (radix + tailwind components) (see https://ui.shadcn.com/docs/installation/vite)

##### Add tailwind.

```bash
pnpm add tailwindcss @tailwindcss/vite
```

##### Replace everything in src/index.css with a tailwind import:

```bash
echo "@import \"tailwindcss\";" > src/index.css
```

##### Edit tsconfig files

The current version of Vite splits TypeScript configuration into three files, two of which need to be edited. Add the baseUrl and paths properties to the compilerOptions section of the tsconfig.json and tsconfig.app.json files:

tsconfig.json

```json
{
  "files": [],
  "references": [
    {
      "path": "./tsconfig.app.json"
    },
    {
      "path": "./tsconfig.node.json"
    }
  ],
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

tsconfig.app.json

```json
{
  "compilerOptions": {
    // ...
    "baseUrl": ".",
    "paths": {
      "@/*": [
        "./src/*"
      ]
    }
    // ...
  }
}
```

##### Update vite.config.ts

```bash
pnpm add -D @types/node
```
Then edit vite.config.ts to include the node types:

```ts
import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
```

##### Run the cli

```bash
pnpm dlx shadcn@latest init
```

##### Add components

```bash
pnpm dlx shadcn@latest add button
pnpm dlx shadcn@latest add alert-dialog
```

You can then use components with `import { Button } from "@/components/ui/button"`

### UI Principles

- Use shadcn AppSideBar as the main navigation for the app: https://ui.shadcn.com/docs/components/sidebar
- Support light and dark mode with Tailwind's built-in dark mode support: https://tailwindcss.com/docs/dark-mode

### App Layout

- The sidebar let's you select which cloud instance you are curently managing from a dropdown.
- The sidebar is divided into Central, Cluster, and Apps.
  - Central: Utilities for managing Wild Central itself.
  - Cluster: Utilities for managing the current Wild Cloud instance.
    - Managing nodes.
    - Managing services.
  - Apps: Managing the apps deployed on the current Wild Cloud instance.
    - List of apps.
    - App details.
    - App logs.
    - App metrics.

