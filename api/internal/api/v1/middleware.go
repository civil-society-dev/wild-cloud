package v1

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

// Context keys for request values.
const (
	InstanceNameKey contextKey = "instanceName"
	AppNameKey      contextKey = "appName"
	ServiceNameKey  contextKey = "serviceName"
	NodeNameKey     contextKey = "nodeName"
)

// ValidateInstanceMiddleware validates that the instance exists and adds the
// instance name to the request context. This eliminates the need for handlers
// to perform the same validation.
//
// For routes without {name} parameter, the middleware passes through without validation.
func (api *API) ValidateInstanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["name"]

		// If no instance name in route, pass through
		if instanceName == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Validate instance exists
		if err := api.instance.ValidateInstance(instanceName); err != nil {
			respondError(w, http.StatusNotFound, "Instance not found")
			return
		}

		// Add instance name to context
		ctx := context.WithValue(r.Context(), InstanceNameKey, instanceName)

		// Also add other route params to context if present
		if appName := vars["app"]; appName != "" {
			ctx = context.WithValue(ctx, AppNameKey, appName)
		}
		if serviceName := vars["service"]; serviceName != "" {
			ctx = context.WithValue(ctx, ServiceNameKey, serviceName)
		}
		if nodeName := vars["node"]; nodeName != "" {
			ctx = context.WithValue(ctx, NodeNameKey, nodeName)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetInstanceName returns the instance name from request context.
// Falls back to mux.Vars if not in context (for backward compatibility during migration).
func GetInstanceName(r *http.Request) string {
	if name, ok := r.Context().Value(InstanceNameKey).(string); ok {
		return name
	}
	return mux.Vars(r)["name"]
}

// GetAppName returns the app name from request context.
// Falls back to mux.Vars if not in context.
func GetAppName(r *http.Request) string {
	if name, ok := r.Context().Value(AppNameKey).(string); ok {
		return name
	}
	return mux.Vars(r)["app"]
}

// GetServiceName returns the service name from request context.
// Falls back to mux.Vars if not in context.
func GetServiceName(r *http.Request) string {
	if name, ok := r.Context().Value(ServiceNameKey).(string); ok {
		return name
	}
	return mux.Vars(r)["service"]
}

// GetNodeName returns the node name from request context.
// Falls back to mux.Vars if not in context.
func GetNodeName(r *http.Request) string {
	if name, ok := r.Context().Value(NodeNameKey).(string); ok {
		return name
	}
	return mux.Vars(r)["node"]
}

// RequireInstanceMiddleware is a stricter version that returns 400 if instance name is missing.
// Use this for routes that absolutely require an instance.
func (api *API) RequireInstanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["name"]

		if instanceName == "" {
			respondError(w, http.StatusBadRequest, "Instance name is required")
			return
		}

		if err := api.instance.ValidateInstance(instanceName); err != nil {
			respondError(w, http.StatusNotFound, "Instance not found")
			return
		}

		ctx := context.WithValue(r.Context(), InstanceNameKey, instanceName)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
