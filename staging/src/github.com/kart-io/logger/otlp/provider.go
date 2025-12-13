package otlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/kart-io/logger/core"
	"github.com/kart-io/logger/option"
	"github.com/kart-io/logger/runtime"
)

// LoggerProvider manages the OTLP logs client for sending logs.
type LoggerProvider struct {
	client   *OTLPClient
	resource *resourcev1.Resource
}

// OTLPClient handles both gRPC and HTTP OTLP logs export.
type OTLPClient struct {
	endpoint string
	protocol string
	timeout  time.Duration
	headers  map[string]string
	insecure bool

	// gRPC client
	grpcConn   *grpc.ClientConn
	grpcClient v1.LogsServiceClient

	// HTTP client
	httpClient *http.Client
}

// NewLoggerProvider creates a new OTLP logger provider.
func NewLoggerProvider(ctx context.Context, opt *option.OTLPOption) (*LoggerProvider, error) {
	if opt == nil || !opt.IsEnabled() {
		return nil, fmt.Errorf("OTLP is not enabled")
	}

	client, err := NewOTLPClient(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP client: %w", err)
	}

	// Get pod/container/hostname based on deployment environment
	podName := runtime.GetPodName("kart-io-service")

	// Create resource with basic infrastructure information only
	// Service info will be added from log attributes when available
	attributes := []*commonv1.KeyValue{
		{
			Key: "pod",
			Value: &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: podName},
			},
		},
		{
			Key: "job",
			Value: &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: "kart-io-logger"},
			},
		},
	}

	// Add namespace if running in Kubernetes
	if runtime.IsKubernetes() {
		if namespace := runtime.GetNamespace(); namespace != "" {
			attributes = append(attributes, &commonv1.KeyValue{
				Key: "ns",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: namespace},
				},
			})
		}
	}

	resource := &resourcev1.Resource{
		Attributes: attributes,
	}

	return &LoggerProvider{
		client:   client,
		resource: resource,
	}, nil
}

// NewOTLPClient creates a new OTLP client.
func NewOTLPClient(opt *option.OTLPOption) (*OTLPClient, error) {
	client := &OTLPClient{
		endpoint: opt.Endpoint,
		protocol: opt.Protocol,
		timeout:  opt.Timeout,
		headers:  opt.Headers,
		insecure: opt.Insecure,
		httpClient: &http.Client{
			Timeout: opt.Timeout,
		},
	}

	if opt.Protocol == "grpc" {
		conn, err := grpc.NewClient(
			opt.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
		}
		client.grpcConn = conn
		client.grpcClient = v1.NewLogsServiceClient(conn)
	}

	return client, nil
}

// SendLogRecord sends a log record via OTLP.
func (p *LoggerProvider) SendLogRecord(level core.Level, message string, attributes map[string]interface{}) error {
	logRecord := p.createLogRecord(level, message, attributes)

	req := &v1.ExportLogsServiceRequest{
		ResourceLogs: []*logsv1.ResourceLogs{
			{
				Resource: p.resource,
				ScopeLogs: []*logsv1.ScopeLogs{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "kart-io/logger",
							Version: "1.0.0",
						},
						LogRecords: []*logsv1.LogRecord{logRecord},
					},
				},
			},
		},
	}

	return p.client.Export(context.Background(), req)
}

// createLogRecord creates an OTLP log record.
func (p *LoggerProvider) createLogRecord(level core.Level, message string, attributes map[string]interface{}) *logsv1.LogRecord {
	now := time.Now()

	// Convert attributes to OTLP format with VictoriaLogs-compatible field names
	otlpAttributes := make([]*commonv1.KeyValue, 0, len(attributes)+10)

	// Extract service information from attributes for VictoriaLogs compatibility
	var serviceName, serviceVersion, instanceName string

	// Try multiple possible service name fields
	if service, exists := attributes["service"]; exists {
		serviceName = fmt.Sprintf("%v", service)
	} else if serviceDotName, exists := attributes["service.name"]; exists {
		serviceName = fmt.Sprintf("%v", serviceDotName)
	} else {
		// Fallback to a default service name if not found
		serviceName = "kart-io-service"
	}

	if version, exists := attributes["version"]; exists {
		serviceVersion = fmt.Sprintf("%v", version)
	} else if serviceDotVersion, exists := attributes["service.version"]; exists {
		serviceVersion = fmt.Sprintf("%v", serviceDotVersion)
	}

	// Get instance name based on deployment environment (same logic as pod name)
	instanceName = runtime.GetPodName("localhost")

	// Add essential VictoriaLogs fields
	otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
		Key: "level", // VictoriaLogs standard field
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: strings.ToLower(level.String())},
		},
	})

	// Add timestamp as string for VictoriaLogs compatibility
	otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
		Key: "@timestamp",
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: now.UTC().Format(time.RFC3339Nano)},
		},
	})

	// Ensure message is also in attributes as _msg (VictoriaLogs standard)
	otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
		Key: "_msg",
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: message},
		},
	})

	// Add service information as attributes (these will appear in the log output)
	if serviceName != "" {
		otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
			Key: "service.name",
			Value: &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: serviceName},
			},
		})

		// Also add as job field for VictoriaLogs stream compatibility
		otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
			Key: "job",
			Value: &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: serviceName},
			},
		})
	}

	if serviceVersion != "" {
		otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
			Key: "service.version",
			Value: &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: serviceVersion},
			},
		})
	}

	// Add instance for VictoriaLogs compatibility
	otlpAttributes = append(otlpAttributes, &commonv1.KeyValue{
		Key: "instance",
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: instanceName},
		},
	})

	// Convert user attributes with proper type handling
	for key, value := range attributes {
		otlpAttr := &commonv1.KeyValue{Key: key}

		switch v := value.(type) {
		case string:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: v},
			}
		case int:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_IntValue{IntValue: int64(v)},
			}
		case int32:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_IntValue{IntValue: int64(v)},
			}
		case int64:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_IntValue{IntValue: v},
			}
		case float32:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_DoubleValue{DoubleValue: float64(v)},
			}
		case float64:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_DoubleValue{DoubleValue: v},
			}
		case bool:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_BoolValue{BoolValue: v},
			}
		case time.Time:
			otlpAttr.Value = &commonv1.AnyValue{
				Value: &commonv1.AnyValue_StringValue{StringValue: v.UTC().Format(time.RFC3339Nano)},
			}
		default:
			// Convert complex types to JSON string
			if jsonBytes, err := json.Marshal(v); err == nil {
				otlpAttr.Value = &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: string(jsonBytes)},
				}
			} else {
				otlpAttr.Value = &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", v)},
				}
			}
		}

		otlpAttributes = append(otlpAttributes, otlpAttr)
	}

	return &logsv1.LogRecord{
		TimeUnixNano:         uint64(now.UnixNano()),
		ObservedTimeUnixNano: uint64(now.UnixNano()),
		SeverityNumber:       mapLevelToSeverityNumber(level),
		SeverityText:         strings.ToUpper(level.String()),
		Body: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: message},
		},
		Attributes: otlpAttributes,
	}
}

// Export exports logs via gRPC or HTTP.
func (c *OTLPClient) Export(ctx context.Context, req *v1.ExportLogsServiceRequest) error {
	if c.protocol == "grpc" {
		return c.exportGRPC(ctx, req)
	}
	return c.exportHTTP(ctx, req)
}

// exportGRPC exports logs via gRPC.
func (c *OTLPClient) exportGRPC(ctx context.Context, req *v1.ExportLogsServiceRequest) error {
	if c.grpcClient == nil {
		return fmt.Errorf("gRPC client not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.Export(ctx, req)
	return err
}

// exportHTTP exports logs via HTTP.
func (c *OTLPClient) exportHTTP(ctx context.Context, req *v1.ExportLogsServiceRequest) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build correct endpoint URL based on endpoint type
	var url string
	if strings.HasPrefix(c.endpoint, "http") {
		// Full URL provided
		url = c.endpoint
	} else {
		// Build URL for OTLP standard endpoints
		// For standard OTLP collectors/agents, use /v1/logs path
		url = fmt.Sprintf("http://%s/v1/logs", c.endpoint)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("User-Agent", "kart-io-logger/1.0.0")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// mapLevelToSeverityNumber maps core.Level to OTLP severity number.
func mapLevelToSeverityNumber(level core.Level) logsv1.SeverityNumber {
	switch level {
	case core.DebugLevel:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case core.InfoLevel:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_INFO
	case core.WarnLevel:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_WARN
	case core.ErrorLevel:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_ERROR
	case core.FatalLevel:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		return logsv1.SeverityNumber_SEVERITY_NUMBER_INFO
	}
}

// Shutdown gracefully shuts down the OTLP client.
func (p *LoggerProvider) Shutdown(ctx context.Context) error {
	if p.client.grpcConn != nil {
		return p.client.grpcConn.Close()
	}
	return nil
}

// ForceFlush forces all pending logs to be sent.
func (p *LoggerProvider) ForceFlush(ctx context.Context) error {
	// Since we're sending logs synchronously, no need to flush
	return nil
}
