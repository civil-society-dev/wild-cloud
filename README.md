# Wild Central API

The Wild Central API is a lightweight service that runs on a local machine (e.g., a Raspberry Pi) to manage Wild Cloud instances on the local network. It provides an interface for users to interact with and manage their Wild Cloud environments.

## Development

Start the development server:

```bash
make dev
```

The API will be available at `http://localhost:5055`.

### Environment Variables

- `WILD_API_DATA_DIR` - Directory for instance data (default: `/var/lib/wild-central`)
- `WILD_DIRECTORY` - Path to Wild Cloud apps directory (default: `/opt/wild-cloud/apps`)
- `WILD_API_DNSMASQ_CONFIG_PATH` - Path to dnsmasq config file (default: `/etc/dnsmasq.d/wild-cloud.conf`)
- `WILD_CORS_ORIGINS` - Comma-separated list of allowed CORS origins for production (default: localhost development origins)

## API Endpoints

The API provides the following endpoint categories:

- **Instances** - Create, list, get, and delete Wild Cloud instances
- **Configuration** - Manage instance config.yaml
- **Secrets** - Manage instance secrets.yaml (redacted by default)
- **Nodes** - Discover, configure, and manage cluster nodes
- **Cluster** - Bootstrap and manage Talos/Kubernetes clusters
- **Services** - Install and manage base infrastructure services
- **Apps** - Deploy and manage Wild Cloud applications
- **PXE** - Manage PXE boot assets for network installation
- **Operations** - Track and stream long-running operations
- **Utilities** - Helper functions and status endpoints
- **dnsmasq** - Configure and manage dnsmasq for network services

See the API handler files in `internal/api/v1/` for detailed endpoint documentation.
