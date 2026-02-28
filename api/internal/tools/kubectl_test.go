package tools

import (
	"testing"
	"time"
)

func TestNewKubectl(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfigPath string
	}{
		{
			name:           "creates Kubectl with kubeconfig path",
			kubeconfigPath: "/path/to/kubeconfig",
		},
		{
			name:           "creates Kubectl with empty path",
			kubeconfigPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := NewKubectl(tt.kubeconfigPath)
			if k == nil {
				t.Fatal("NewKubectl() returned nil")
			}
			if k.kubeconfigPath != tt.kubeconfigPath {
				t.Errorf("kubeconfigPath = %q, want %q", k.kubeconfigPath, tt.kubeconfigPath)
			}
		})
	}
}

func TestKubectlDeploymentExists(t *testing.T) {
	tests := []struct {
		name      string
		depName   string
		namespace string
		skipTest  bool
	}{
		{
			name:      "check deployment exists",
			depName:   "test-deployment",
			namespace: "default",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			exists := k.DeploymentExists(tt.depName, tt.namespace)
			_ = exists // Result depends on actual cluster state
		})
	}
}

func TestKubectlGetPods(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		detailed  bool
		skipTest  bool
	}{
		{
			name:      "get pods basic",
			namespace: "default",
			detailed:  false,
			skipTest:  true,
		},
		{
			name:      "get pods detailed",
			namespace: "kube-system",
			detailed:  true,
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			pods, err := k.GetPods(tt.namespace, tt.detailed)

			if err == nil {
				if pods == nil {
					t.Error("GetPods() returned nil slice without error")
				}
				// Verify pod structure
				for i, pod := range pods {
					if pod.Name == "" {
						t.Errorf("pod[%d].Name is empty", i)
					}
					if pod.Status == "" {
						t.Errorf("pod[%d].Status is empty", i)
					}
					if pod.Ready == "" {
						t.Errorf("pod[%d].Ready is empty", i)
					}
					if pod.Age == "" {
						t.Errorf("pod[%d].Age is empty", i)
					}
					if tt.detailed && pod.Containers == nil {
						t.Errorf("pod[%d].Containers is nil in detailed mode", i)
					}
				}
			}
		})
	}
}

func TestKubectlGetFirstPodName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		skipTest  bool
	}{
		{
			name:      "get first pod name",
			namespace: "kube-system",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			podName, err := k.GetFirstPodName(tt.namespace)

			if err == nil {
				if podName == "" {
					t.Error("GetFirstPodName() returned empty string without error")
				}
			}
		})
	}
}

func TestKubectlGetPodContainers(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		podName   string
		skipTest  bool
	}{
		{
			name:      "get pod containers",
			namespace: "kube-system",
			podName:   "coredns-123",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			containers, err := k.GetPodContainers(tt.namespace, tt.podName)

			if err == nil {
				if containers == nil {
					t.Error("GetPodContainers() returned nil slice without error")
				}
			}
		})
	}
}

func TestKubectlGetDeployment(t *testing.T) {
	tests := []struct {
		name      string
		depName   string
		namespace string
		skipTest  bool
	}{
		{
			name:      "get deployment info",
			depName:   "test-deployment",
			namespace: "default",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			depInfo, err := k.GetDeployment(tt.depName, tt.namespace)

			if err == nil {
				if depInfo == nil {
					t.Fatal("GetDeployment() returned nil without error")
				}
				// Desired should be non-negative
				if depInfo.Desired < 0 {
					t.Errorf("Desired = %d, should be non-negative", depInfo.Desired)
				}
			}
		})
	}
}

func TestKubectlGetReplicas(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		skipTest  bool
	}{
		{
			name:      "get replicas for namespace",
			namespace: "default",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			replicaInfo, err := k.GetReplicas(tt.namespace)

			if err == nil {
				if replicaInfo == nil {
					t.Fatal("GetReplicas() returned nil without error")
				}
				// All values should be non-negative
				if replicaInfo.Desired < 0 {
					t.Error("Desired < 0")
				}
				if replicaInfo.Current < 0 {
					t.Error("Current < 0")
				}
				if replicaInfo.Ready < 0 {
					t.Error("Ready < 0")
				}
				if replicaInfo.Available < 0 {
					t.Error("Available < 0")
				}
			}
		})
	}
}

func TestKubectlGetResources(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		skipTest  bool
	}{
		{
			name:      "get resources for namespace",
			namespace: "default",
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			usage, err := k.GetResources(tt.namespace)

			if err == nil {
				if usage == nil {
					t.Error("GetResources() returned nil without error")
				}
			}
		})
	}
}

func TestKubectlGetRecentEvents(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		limit     int
		skipTest  bool
	}{
		{
			name:      "get recent events",
			namespace: "default",
			limit:     10,
			skipTest:  true,
		},
		{
			name:      "get all events with zero limit",
			namespace: "default",
			limit:     0,
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			events, err := k.GetRecentEvents(tt.namespace, tt.limit)

			if err == nil {
				if events == nil {
					t.Error("GetRecentEvents() returned nil slice without error")
				}
				if tt.limit > 0 && len(events) > tt.limit {
					t.Errorf("len(events) = %d, want <= %d", len(events), tt.limit)
				}
			}
		})
	}
}

func TestKubectlGetLogs(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		podName   string
		opts      LogOptions
		skipTest  bool
	}{
		{
			name:      "get logs with tail",
			namespace: "kube-system",
			podName:   "coredns-123",
			opts:      LogOptions{Tail: 100},
			skipTest:  true,
		},
		{
			name:      "get logs with container",
			namespace: "kube-system",
			podName:   "coredns-123",
			opts:      LogOptions{Container: "coredns", Tail: 50},
			skipTest:  true,
		},
		{
			name:      "get previous logs",
			namespace: "default",
			podName:   "test-pod",
			opts:      LogOptions{Previous: true, Tail: 100},
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			logs, err := k.GetLogs(tt.namespace, tt.podName, tt.opts)

			if err == nil {
				if logs == nil {
					t.Error("GetLogs() returned nil slice without error")
				}
			}
		})
	}
}

func TestKubectlStreamLogs(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		podName   string
		opts      LogOptions
		skipTest  bool
	}{
		{
			name:      "stream logs",
			namespace: "default",
			podName:   "test-pod",
			opts:      LogOptions{Tail: 10},
			skipTest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			cmd, err := k.StreamLogs(tt.namespace, tt.podName, tt.opts)

			if err == nil {
				if cmd == nil {
					t.Error("StreamLogs() returned nil command without error")
				}
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			want:     "5m",
		},
		{
			name:     "hours",
			duration: 3 * time.Hour,
			want:     "3h",
		},
		{
			name:     "days",
			duration: 48 * time.Hour,
			want:     "2d",
		},
		{
			name:     "less than minute",
			duration: 30 * time.Second,
			want:     "30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.duration)
			if got != tt.want {
				t.Errorf("formatAge(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestParseCPUQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity string
		want     int64
	}{
		{
			name:     "millicores",
			quantity: "500m",
			want:     500,
		},
		{
			name:     "cores as plain number",
			quantity: "2",
			want:     2000,
		},
		{
			name:     "fractional cores",
			quantity: "0.5",
			want:     500,
		},
		{
			name:     "empty string",
			quantity: "",
			want:     0,
		},
		{
			name:     "whitespace",
			quantity: "  ",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCPUQuantity(tt.quantity)
			if got != tt.want {
				t.Errorf("parseCPUQuantity(%q) = %d, want %d", tt.quantity, got, tt.want)
			}
		})
	}
}

func TestParseMemoryQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity string
		want     int64
	}{
		{
			name:     "Ki suffix",
			quantity: "100Ki",
			want:     100 * 1024,
		},
		{
			name:     "Mi suffix",
			quantity: "512Mi",
			want:     512 * 1024 * 1024,
		},
		{
			name:     "Gi suffix",
			quantity: "2Gi",
			want:     2 * 1024 * 1024 * 1024,
		},
		{
			name:     "K suffix",
			quantity: "100K",
			want:     100 * 1000,
		},
		{
			name:     "M suffix",
			quantity: "500M",
			want:     500 * 1000 * 1000,
		},
		{
			name:     "G suffix",
			quantity: "1G",
			want:     1 * 1000 * 1000 * 1000,
		},
		{
			name:     "plain bytes",
			quantity: "1048576",
			want:     1048576,
		},
		{
			name:     "empty string",
			quantity: "",
			want:     0,
		},
		{
			name:     "whitespace",
			quantity: "  ",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMemoryQuantity(tt.quantity)
			if got != tt.want {
				t.Errorf("parseMemoryQuantity(%q) = %d, want %d", tt.quantity, got, tt.want)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name       string
		millicores int64
		want       string
	}{
		{
			name:       "zero",
			millicores: 0,
			want:       "0",
		},
		{
			name:       "millicores",
			millicores: 500,
			want:       "500m",
		},
		{
			name:       "one core",
			millicores: 1000,
			want:       "1.0",
		},
		{
			name:       "two and half cores",
			millicores: 2500,
			want:       "2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCPU(tt.millicores)
			if got != tt.want {
				t.Errorf("formatCPU(%d) = %q, want %q", tt.millicores, got, tt.want)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero",
			bytes: 0,
			want:  "0",
		},
		{
			name:  "bytes",
			bytes: 512,
			want:  "512B",
		},
		{
			name:  "kibibytes",
			bytes: 1024,
			want:  "1.0Ki",
		},
		{
			name:  "mebibytes",
			bytes: 1024 * 1024,
			want:  "1.0Mi",
		},
		{
			name:  "gibibytes",
			bytes: 2 * 1024 * 1024 * 1024,
			want:  "2.0Gi",
		},
		{
			name:  "tebibytes",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.0Ti",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMemory(tt.bytes)
			if got != tt.want {
				t.Errorf("formatMemory(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestKubectlGetPodsByDeployment(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		deploymentName string
		skipTest       bool
		wantError      bool
	}{
		{
			name:           "get pods for deployment",
			namespace:      "kube-system",
			deploymentName: "coredns",
			skipTest:       true,
			wantError:      false,
		},
		{
			name:           "deployment not found",
			namespace:      "default",
			deploymentName: "nonexistent-deployment",
			skipTest:       true,
			wantError:      true,
		},
		{
			name:           "empty deployment name",
			namespace:      "default",
			deploymentName: "",
			skipTest:       true,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			pods, err := k.GetPodsByDeployment(tt.namespace, tt.deploymentName)

			if tt.wantError {
				if err == nil {
					t.Error("GetPodsByDeployment() error = nil, wantError = true")
				}
			} else {
				if err != nil {
					t.Errorf("GetPodsByDeployment() error = %v, wantError = false", err)
				}
				if pods == nil {
					t.Error("GetPodsByDeployment() returned nil slice without error")
				}
			}
		})
	}
}

func TestKubectlGetPodsByLabel(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		labelSelector string
		skipTest      bool
		wantError     bool
	}{
		{
			name:          "get pods by label",
			namespace:     "kube-system",
			labelSelector: "k8s-app=kube-dns",
			skipTest:      true,
			wantError:     false,
		},
		{
			name:          "get all pods with empty selector",
			namespace:     "default",
			labelSelector: "",
			skipTest:      true,
			wantError:     false,
		},
		{
			name:          "get pods by multiple labels",
			namespace:     "default",
			labelSelector: "app=nginx,environment=production",
			skipTest:      true,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires kubectl and running cluster")
			}

			k := NewKubectl("")
			pods, err := k.GetPodsByLabel(tt.namespace, tt.labelSelector)

			if tt.wantError {
				if err == nil {
					t.Error("GetPodsByLabel() error = nil, wantError = true")
				}
			} else {
				if err != nil {
					t.Errorf("GetPodsByLabel() error = %v, wantError = false", err)
				}
				if pods == nil {
					t.Error("GetPodsByLabel() returned nil slice without error")
				}
				// Verify pod structure
				for i, pod := range pods {
					if pod.Name == "" {
						t.Errorf("pod[%d].Name is empty", i)
					}
					if pod.Status == "" {
						t.Errorf("pod[%d].Status is empty", i)
					}
				}
			}
		})
	}
}

