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

### UX Principles

These principles ensure consistent, intuitive interfaces that align with Wild Cloud's philosophy of simplicity and clarity. Use them as quality control when building new components.

#### Navigation & Structure

- **Use shadcn AppSideBar** as the main navigation: https://ui.shadcn.com/docs/components/sidebar
- **Card-Based Layout**: Group related content in Card components
  - Primary cards: `p-6` padding
  - Nested cards: `p-4` padding with subtle shadows
  - Use cards to create visual hierarchy through nesting
- **Spacing Rhythm**: Maintain consistent vertical spacing
  - Major sections: `space-y-6`
  - Related items: `space-y-4`
  - Form fields: `space-y-3`
  - Inline elements: `gap-2`, `gap-3`, or `gap-4`

#### Visual Design

- **Dark Mode**: Support both light and dark modes using Tailwind's `dark:` prefix
  - Test all components in both modes for contrast and readability
  - Use semantic color tokens that adapt to theme
- **Status Color System**: Use semantic left border colors to categorize content
  - Blue (`border-l-blue-500`): Configuration sections
  - Green (`border-l-green-500`): Network/infrastructure
  - Red (`border-l-red-500`): Errors and warnings
  - Cyan: Educational content
- **Icon-Text Pairing**: Pair important text with Lucide icons
  - Place icons in colored containers: `p-2 bg-primary/10 rounded-lg`
  - Provides visual anchors and improves scannability
- **Technical Data Display**: Show technical information clearly
  - Use `font-mono` class for IPs, domains, configuration values
  - Display in `bg-muted rounded-md p-2` containers

#### Component Patterns

- **Edit/View Mode Toggle**: For configuration sections
  - Read-only: Display in `bg-muted rounded-md font-mono` containers with Edit button
  - Edit mode: Replace with form inputs in-place
  - Provides lightweight editing without context switching
- **Drawers for Complex Forms**: Use side panels for detailed input
  - Maintains context with main content
  - Better than modals for forms that benefit from seeing related data
- **Educational Content**: Use gradient cards for helpful information
  - Background: `from-cyan-50 to-blue-50 dark:from-cyan-900/20 dark:to-blue-900/20`
  - Include book icon and clear, concise guidance
  - Makes learning feel integrated, not intrusive
- **Empty States**: Center content with clear next actions
  - Large icon: `h-12 w-12 text-muted-foreground`
  - Descriptive title and explanation
  - Suggest action to resolve empty state

#### Section Headers

Structure all major section headers consistently:

```tsx
<div className="flex items-center gap-4 mb-6">
  <div className="p-2 bg-primary/10 rounded-lg">
    <IconComponent className="h-6 w-6 text-primary" />
  </div>
  <div>
    <h2 className="text-2xl font-semibold">Section Title</h2>
    <p className="text-muted-foreground">
      Brief description of section purpose
    </p>
  </div>
</div>
```

#### Status & Feedback

- **Status Badges**: Use colored badges with icons for state indication
  - Keep compact but descriptive
  - Include hover/expansion for additional detail
- **Alert Positioning**: Place alerts near related content
  - Use semantic colors and icons (CheckCircle, AlertCircle, XCircle)
  - Include dismissible X button for manual dismissal
- **Success Messages**: Auto-dismiss after 5 seconds
  - Green color with CheckCircle icon
  - Clear, affirmative message
- **Error Messages**: Structured and actionable
  - Title in bold, detailed message below
  - Red color with AlertCircle icon
  - Suggest resolution when possible
- **Loading States**: Context-appropriate indicators
  - Inline: Use `Loader2` spinner in buttons/actions
  - Full section: Card with centered spinner and descriptive text

#### Form Components

Use react-hook-form for all forms. Never duplicate component styling.

**Standard Form Pattern**:
```tsx
import { useForm, Controller } from 'react-hook-form';
import { Input, Label, Button } from '@/components/ui';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';

const { register, handleSubmit, control, formState: { errors } } = useForm({
  defaultValues: { /* ... */ }
});

<form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
  <div>
    <Label htmlFor="text">Text Field</Label>
    <Input {...register('text', { required: 'Required' })} className="mt-1" />
    {errors.text && <p className="text-sm text-red-600 mt-1">{errors.text.message}</p>}
  </div>

  <div>
    <Label htmlFor="select">Select Field</Label>
    <Controller
      name="select"
      control={control}
      rules={{ required: 'Required' }}
      render={({ field }) => (
        <Select value={field.value} onValueChange={field.onChange}>
          <SelectTrigger className="mt-1">
            <SelectValue placeholder="Choose..." />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="option1">Option 1</SelectItem>
            <SelectItem value="option2">Option 2</SelectItem>
          </SelectContent>
        </Select>
      )}
    />
    {errors.select && <p className="text-sm text-red-600 mt-1">{errors.select.message}</p>}
  </div>
</form>
```

**Rules**:
- **Text inputs**: Use `Input` with `register()`
- **Select dropdowns**: Use `Select` components with `Controller` (never native `<select>`)
- **All labels**: Use `Label` with `htmlFor` attribute
- **Never copy classes**: Components provide default styling, only add spacing like `mt-1`
- **Form spacing**: `space-y-3` on form containers
- **Error messages**: `text-sm text-red-600 mt-1`
- **Multi-action forms**: Place buttons side-by-side with `flex gap-2`

#### Accessibility

- **Focus Indicators**: All interactive elements must have visible focus states
  - Use consistent `focus-visible:ring-*` styles
  - Test keyboard navigation on all new components
- **Screen Reader Support**: Proper semantic HTML and ARIA labels
  - Use Label components for form inputs
  - Provide descriptive text for icon-only buttons
  - Test with screen readers when adding complex interactions

#### Progressive Disclosure

- **Just-in-Time Information**: Start simple, reveal details on demand
  - Summary view by default
  - Details through drawers, accordions, or inline expansion
  - Reduces initial cognitive load
- **Educational Context**: Provide help without interrupting flow
  - Use gradient educational cards in logical places
  - Include "Learn more" links to external documentation
  - Keep content concise and actionable

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

