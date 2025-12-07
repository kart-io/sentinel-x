package runtime

import (
	"os"
	"strings"
)

// GetPodName determines the pod/container/host name based on deployment environment.
// It detects the runtime environment and returns the appropriate identifier:
// - Kubernetes: pod name or node name
// - Docker/Container: container name or hostname
// - Host: system hostname
// - Fallback: provided fallback name
func GetPodName(fallbackName string) string {
	// Priority 1: Kubernetes pod name (most specific)
	if podName := os.Getenv("POD_NAME"); podName != "" {
		return podName
	}

	// Priority 2: Kubernetes node name
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		return nodeName
	}

	// Priority 3: Docker/container hostname
	if containerName := os.Getenv("HOSTNAME"); containerName != "" {
		return containerName
	}

	// Priority 4: Container name from Docker
	if containerName := os.Getenv("CONTAINER_NAME"); containerName != "" {
		return containerName
	}

	// Priority 5: System hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}

	// Fallback: use service name
	return fallbackName
}

// IsKubernetes detects if the application is running in a Kubernetes environment.
// It checks for Kubernetes-specific environment variables that are typically
// set by the kubelet or Kubernetes deployment manifests.
func IsKubernetes() bool {
	// Check for Kubernetes-specific environment variables
	return os.Getenv("POD_NAME") != "" ||
		os.Getenv("POD_NAMESPACE") != "" ||
		os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// GetNamespace returns the Kubernetes namespace the application is running in.
// It attempts multiple methods to determine the namespace:
// - Environment variables (POD_NAMESPACE, NAMESPACE)
// - Service account namespace file
// - Default namespace as fallback
func GetNamespace() string {
	// Priority 1: POD_NAMESPACE environment variable
	if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
		return namespace
	}

	// Priority 2: Read from service account token file
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		namespace := strings.TrimSpace(string(data))
		if namespace != "" {
			return namespace
		}
	}

	// Priority 3: NAMESPACE environment variable
	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		return namespace
	}

	// Fallback: default namespace
	return "default"
}
