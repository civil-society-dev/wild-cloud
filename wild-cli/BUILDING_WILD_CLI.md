## Dev Environment Requirements

- Go 1.21+
- GNU Make (for build automation)

## Patterns

### Instance-scoped Commands

CLI commands that operate on a specific Wild Cloud instance should follow this pattern:

```go
// In cmd/utility.go
var dashboardTokenCmd = &cobra.Command{
    Use:   "token",
    Short: "Get dashboard token",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Get instance from CLI context
        instanceName, err := getInstanceName()
        if err != nil {
            return err
        }

        // 2. Call instance-scoped API endpoint with instance in URL
        resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/utilities/dashboard/token", instanceName))
        if err != nil {
            return err
        }

        // 3. Process response
        data := resp.GetMap("data")
        if data == nil {
            return fmt.Errorf("no data in response")
        }

        token, ok := data["token"].(string)
        if !ok {
            return fmt.Errorf("no token in response")
        }

        // 4. Display result
        fmt.Println(token)
        return nil
    },
}
```

#### Key Principles

1. **Get instance from context**: Use `getInstanceName()` to get the current instance from CLI context
2. **Instance in URL path**: Include the instance name in the API endpoint URL path
3. **Stateless API calls**: Don't rely on server-side session state - pass instance explicitly
4. **Handle errors gracefully**: Return clear error messages if instance is not set or API call fails
