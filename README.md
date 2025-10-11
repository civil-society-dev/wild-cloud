# Wild Central Daemon

The Wild Central Daemon is a lightweight service that runs on a local machine (e.g., a Raspberry Pi) to manage Wild Cloud instances on the local network. It provides an interface for users to interact with and manage their Wild Cloud environments.

## Development

```bash
make dev
```

## Usage

### Batch Configuration Update Endpoint

#### Overview

The batch configuration update endpoint allows updating multiple configuration values in a single atomic request.

#### Endpoint

```
PATCH /api/v1/instances/{name}/config
```

#### Request Format

```json
{
  "updates": [
    {"path": "string", "value": "any"},
    {"path": "string", "value": "any"}
  ]
}
```

#### Response Format

Success (200 OK):
```json
{
  "message": "Configuration updated successfully",
  "updated": 3
}
```

Error (400 Bad Request / 404 Not Found / 500 Internal Server Error):
```json
{
  "error": "error message"
}
```

#### Usage Examples

##### Example 1: Update Basic Configuration Values

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/my-cloud/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"path": "baseDomain", "value": "example.com"},
      {"path": "domain", "value": "wild.example.com"},
      {"path": "internalDomain", "value": "int.wild.example.com"}
    ]
  }'
```

Response:
```json
{
  "message": "Configuration updated successfully",
  "updated": 3
}
```

##### Example 2: Update Nested Configuration Values

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/my-cloud/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"path": "cluster.name", "value": "prod-cluster"},
      {"path": "cluster.loadBalancerIp", "value": "192.168.1.100"},
      {"path": "cluster.ipAddressPool", "value": "192.168.1.100-192.168.1.200"}
    ]
  }'
```

##### Example 3: Update Array Values

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/my-cloud/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"path": "cluster.nodes.activeNodes[0]", "value": "node-1"},
      {"path": "cluster.nodes.activeNodes[1]", "value": "node-2"}
    ]
  }'
```

##### Example 4: Error Handling - Invalid Instance

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/nonexistent/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"path": "baseDomain", "value": "example.com"}
    ]
  }'
```

Response (404):
```json
{
  "error": "Instance not found: instance nonexistent does not exist"
}
```

##### Example 5: Error Handling - Empty Updates

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/my-cloud/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": []
  }'
```

Response (400):
```json
{
  "error": "updates array is required and cannot be empty"
}
```

##### Example 6: Error Handling - Missing Path

```bash
curl -X PATCH http://localhost:8080/api/v1/instances/my-cloud/config \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"path": "", "value": "example.com"}
    ]
  }'
```

Response (400):
```json
{
  "error": "update[0]: path is required"
}
```

#### Configuration Path Syntax

The `path` field uses YAML path syntax as implemented by the `yq` tool:

- Simple fields: `baseDomain`
- Nested fields: `cluster.name`
- Array elements: `cluster.nodes.activeNodes[0]`
- Array append: `cluster.nodes.activeNodes[+]`

Refer to the yq documentation for advanced path syntax.
