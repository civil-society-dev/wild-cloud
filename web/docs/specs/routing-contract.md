# Wild Cloud Web App Routing Contract

## Front Matter

**Module Name**: `routing`
**Module Type**: Infrastructure
**Version**: 1.0.0
**Status**: Draft
**Last Updated**: 2025-10-12
**Owner**: Wild Cloud Development Team

### Dependencies

- `react`: ^19.1.0
- `react-router`: ^7.0.0 (to be added)

### Consumers

- All page components within Wild Cloud Web App
- Navigation components (AppSidebar)
- Instance context management
- External navigation links

## Purpose

This module defines the routing system for the Wild Cloud web application, providing declarative navigation between pages, URL-based state management, and integration with the existing instance context system.

## Public API

### Router Configuration

The routing system provides a centralized router configuration that manages all application routes.

#### Primary Routes

```typescript
interface RouteDefinition {
  path: string;
  element: React.ComponentType;
  loader?: () => Promise<unknown>;
  errorElement?: React.ComponentType;
}
```

**Root Route**:
- **Path**: `/`
- **Purpose**: Landing page and instance selector
- **Authentication**: None required
- **Data Loading**: None

**Instance Routes**:
- **Path Pattern**: `/instances/:instanceId/*`
- **Purpose**: All instance-specific pages
- **Authentication**: Instance must exist
- **Data Loading**: Instance configuration loaded at this level

#### Instance-Scoped Routes

All routes under `/instances/:instanceId/`:

| Path | Purpose | Data Dependencies |
|------|---------|-------------------|
| `dashboard` | Overview and quick status | Instance config, cluster status |
| `operations` | Operation monitoring and logs | Active operations list |
| `cluster/health` | Cluster health metrics | Node status, etcd health |
| `cluster/access` | Kubeconfig/Talosconfig download | Instance credentials |
| `secrets` | Secrets management interface | Instance secrets (redacted) |
| `services` | Base services management | Installed services list |
| `utilities` | Utilities panel | Available utilities |
| `cloud` | Cloud configuration | Cloud settings |
| `dns` | DNS management | DNS configuration |
| `dhcp` | DHCP management | DHCP configuration |
| `pxe` | PXE boot configuration | PXE assets and config |
| `infrastructure` | Cluster nodes | Node list and status |
| `cluster` | Cluster services | Kubernetes services |
| `apps` | Application management | Installed apps |
| `advanced` | Advanced settings | System configuration |

### Navigation Hooks

#### useNavigate

```typescript
function useNavigate(): NavigateFunction;

interface NavigateFunction {
  (to: string | number, options?: NavigateOptions): void;
}

interface NavigateOptions {
  replace?: boolean;
  state?: unknown;
}
```

**Purpose**: Programmatic navigation within the application.

**Examples**:
```typescript
// Navigate to a specific instance dashboard
navigate('/instances/prod-cluster/dashboard');

// Navigate back
navigate(-1);

// Replace current history entry
navigate('/instances/prod-cluster/operations', { replace: true });
```

**Error Conditions**:
- Invalid paths are handled by error boundary
- Missing instance IDs redirect to root

#### useParams

```typescript
function useParams<T extends Record<string, string>>(): T;
```

**Purpose**: Access URL parameters, primarily for extracting instance ID.

**Example**:
```typescript
const { instanceId } = useParams<{ instanceId: string }>();
```

**Error Conditions**:
- Returns undefined for missing parameters
- Type safety through TypeScript generics

#### useLocation

```typescript
interface Location {
  pathname: string;
  search: string;
  hash: string;
  state: unknown;
  key: string;
}

function useLocation(): Location;
```

**Purpose**: Access current location information for conditional rendering or analytics.

#### useSearchParams

```typescript
function useSearchParams(): [
  URLSearchParams,
  (nextInit: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams)) => void
];
```

**Purpose**: Read and write URL query parameters for filters, sorting, and view state.

**Example**:
```typescript
const [searchParams, setSearchParams] = useSearchParams();
const view = searchParams.get('view') || 'grid';
setSearchParams({ view: 'list' });
```

### Link Component

```typescript
interface LinkProps {
  to: string;
  replace?: boolean;
  state?: unknown;
  children: React.ReactNode;
  className?: string;
}

function Link(props: LinkProps): JSX.Element;
```

**Purpose**: Declarative navigation component for user-triggered navigation.

**Example**:
```typescript
<Link to="/instances/prod-cluster/dashboard">
  Go to Dashboard
</Link>
```

**Behavior**:
- Prevents default browser navigation
- Supports keyboard navigation (Enter)
- Maintains browser history
- Supports Ctrl/Cmd+Click to open in new tab

### NavLink Component

```typescript
interface NavLinkProps extends LinkProps {
  caseSensitive?: boolean;
  end?: boolean;
  className?: string | ((props: { isActive: boolean; isPending: boolean }) => string);
  style?: React.CSSProperties | ((props: { isActive: boolean; isPending: boolean }) => React.CSSProperties);
}

function NavLink(props: NavLinkProps): JSX.Element;
```

**Purpose**: Navigation links that are aware of their active state.

**Example**:
```typescript
<NavLink
  to="/instances/prod-cluster/dashboard"
  className={({ isActive }) => isActive ? 'active-nav-link' : 'nav-link'}
>
  Dashboard
</NavLink>
```

## Data Models

### Route Parameters

```typescript
interface InstanceRouteParams {
  instanceId: string;
}
```

**Field Specifications**:
- `instanceId`: String identifier for the instance
  - **Required**: Yes
  - **Format**: Alphanumeric with hyphens, 1-64 characters
  - **Validation**: Must correspond to an existing instance
  - **Example**: `"prod-cluster"`, `"staging-env"`

### Navigation State

```typescript
interface NavigationState {
  from?: string;
  returnTo?: string;
  [key: string]: unknown;
}
```

**Purpose**: Preserve state across navigation, such as return URLs or form data.

**Example**:
```typescript
navigate('/instances/prod-cluster/secrets', {
  state: { returnTo: '/instances/prod-cluster/dashboard' }
});
```

## Error Model

### Route Errors

| Error Code | Condition | User Impact | Recovery Action |
|------------|-----------|-------------|-----------------|
| `ROUTE_NOT_FOUND` | Path does not match any route | 404 page displayed | Redirect to root or show available routes |
| `INSTANCE_NOT_FOUND` | Instance ID in URL does not exist | Error boundary with message | Redirect to instance selector at `/` |
| `INVALID_INSTANCE_ID` | Instance ID format invalid | Validation error displayed | Show error message, redirect to `/` |
| `NAVIGATION_CANCELLED` | User cancelled pending navigation | No visible change | Continue at current route |
| `LOADER_ERROR` | Route data loader failed | Error boundary with retry option | Show error, allow retry or navigate away |

### Error Response Format

```typescript
interface RouteError {
  code: string;
  message: string;
  status?: number;
  cause?: Error;
}
```

**Example**:
```typescript
{
  code: "INSTANCE_NOT_FOUND",
  message: "Instance 'unknown-instance' does not exist",
  status: 404
}
```

## Performance Characteristics

### Route Transition Times

- **Static Routes**: < 50ms (no data loading)
- **Instance Routes**: < 200ms (with instance config loading)
- **Heavy Data Routes**: < 500ms (with large data sets)

### Bundle Size

- **Router Core**: ~45KB (minified)
- **Navigation Components**: ~5KB
- **Per-Route Code Splitting**: Enabled by default

### Memory Usage

- **History Stack**: O(n) where n is number of navigation entries
- **Route Cache**: Configurable, default 10 entries
- **Cleanup**: Automatic on unmount

## Configuration Requirements

### Environment Variables

None required. All routing is handled client-side.

### Route Configuration

```typescript
interface RouterConfig {
  basename?: string;
  future?: {
    v7_startTransition?: boolean;
    v7_relativeSplatPath?: boolean;
  };
}
```

**basename**: Optional base path for deployment in subdirectories
- **Default**: `"/"`
- **Example**: `"/app"` if deployed to `example.com/app/`

## Conformance Criteria

### Functional Requirements

1. **F-ROUTE-01**: Router SHALL render correct component for each defined route
2. **F-ROUTE-02**: Router SHALL extract instance ID from URL parameters
3. **F-ROUTE-03**: Navigation hooks SHALL update browser history
4. **F-ROUTE-04**: Back/forward browser buttons SHALL work correctly
5. **F-ROUTE-05**: Invalid routes SHALL display error boundary
6. **F-ROUTE-06**: Instance routes SHALL validate instance existence
7. **F-ROUTE-07**: Link components SHALL support keyboard navigation
8. **F-ROUTE-08**: Routes SHALL support lazy loading of components

### Non-Functional Requirements

1. **NF-ROUTE-01**: Route transitions SHALL complete in < 200ms for cached routes
2. **NF-ROUTE-02**: Router SHALL support browser back/forward without page reload
3. **NF-ROUTE-03**: Navigation SHALL preserve scroll position when appropriate
4. **NF-ROUTE-04**: Router SHALL be compatible with React 19.1+
5. **NF-ROUTE-05**: Routes SHALL be defined declaratively in configuration
6. **NF-ROUTE-06**: Router SHALL integrate with existing ErrorBoundary
7. **NF-ROUTE-07**: Navigation SHALL work with InstanceContext

### Integration Requirements

1. **I-ROUTE-01**: Router SHALL integrate with InstanceContext
2. **I-ROUTE-02**: AppSidebar SHALL use routing for navigation
3. **I-ROUTE-03**: All page components SHALL be routed
4. **I-ROUTE-04**: Router SHALL integrate with React Query for data loading
5. **I-ROUTE-05**: Router SHALL support Vite code splitting

## API Stability

**Versioning Scheme**: Semantic Versioning (SemVer)

**Stability Level**: Stable (1.0.0+)

**Breaking Changes**:
- Route path changes require major version bump
- Hook signature changes require major version bump
- Added routes or optional parameters are minor version bumps

**Deprecation Policy**:
- Deprecated routes supported for 2 minor versions
- Console warnings for deprecated route usage
- Migration guide provided for breaking changes

## Security Considerations

### Route Protection

Instance routes SHALL verify:
1. Instance ID exists in available instances
2. User has permission to access instance (future)

### XSS Prevention

- All route parameters SHALL be sanitized
- User-provided navigation state SHALL be validated
- No executable code in route parameters

### CSRF Protection

Not applicable - all navigation is client-side without authentication tokens.

## Browser Compatibility

**Supported Browsers**:
- Chrome/Edge: Last 2 versions
- Firefox: Last 2 versions
- Safari: Last 2 versions

**Required Browser APIs**:
- History API
- URL API
- ES6+ JavaScript features

## Examples

### Basic Navigation

```typescript
// In a component
import { useNavigate } from 'react-router';

function InstanceCard({ instance }) {
  const navigate = useNavigate();

  const handleClick = () => {
    navigate(`/instances/${instance.id}/dashboard`);
  };

  return <button onClick={handleClick}>Open {instance.name}</button>;
}
```

### Using Route Parameters

```typescript
// In an instance page
import { useParams } from 'react-router';
import { useInstanceContext } from '../hooks';

function DashboardPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { setCurrentInstance } = useInstanceContext();

  useEffect(() => {
    setCurrentInstance(instanceId);
  }, [instanceId, setCurrentInstance]);

  return <div>Dashboard for {instanceId}</div>;
}
```

### Sidebar Integration

```typescript
// AppSidebar using NavLink
import { NavLink } from 'react-router';

function AppSidebar() {
  const { instanceId } = useParams();

  return (
    <nav>
      <NavLink
        to={`/instances/${instanceId}/dashboard`}
        className={({ isActive }) => isActive ? 'active' : ''}
      >
        Dashboard
      </NavLink>
    </nav>
  );
}
```

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-10-12 | Initial contract definition |
