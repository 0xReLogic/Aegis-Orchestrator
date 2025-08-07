package health

import (
	"context"
	"fmt"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
	"github.com/0xReLogic/Aegis-Orchestrator/pkg/plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// GRPCHealthChecker implements a health checker for gRPC endpoints
type GRPCHealthChecker struct {
	name      string
	endpoint  string
	timeout   time.Duration
	service   string
	conn      *grpc.ClientConn
	useHealth bool
}

// NewGRPCHealthChecker creates a new gRPC health checker
func NewGRPCHealthChecker(name, endpoint string, timeout time.Duration) *GRPCHealthChecker {
	return &GRPCHealthChecker{
		name:      name,
		endpoint:  endpoint,
		timeout:   timeout,
		service:   "",
		useHealth: true,
	}
}

// Name returns the name of the health checker
func (g *GRPCHealthChecker) Name() string {
	return g.name
}

// Type returns the type of the health checker
func (g *GRPCHealthChecker) Type() string {
	return "grpc"
}

// Configure configures the health checker with the given options
func (g *GRPCHealthChecker) Configure(options map[string]interface{}) error {
	if endpoint, ok := options["endpoint"].(string); ok {
		g.endpoint = endpoint
	}

	if timeout, ok := options["timeout"].(time.Duration); ok {
		g.timeout = timeout
	}

	if service, ok := options["service"].(string); ok {
		g.service = service
	}

	if useHealth, ok := options["useHealth"].(bool); ok {
		g.useHealth = useHealth
	}

	return nil
}

// Check performs a health check and returns the result
func (g *GRPCHealthChecker) Check() (*plugin.HealthCheckResult, error) {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// Create connection options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	// Connect to the gRPC server
	conn, err := grpc.DialContext(ctx, g.endpoint, opts...)
	if err != nil {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("Failed to connect to gRPC server: %v", err),
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
			},
		}, nil
	}
	defer conn.Close()

	// Check connection state
	state := conn.GetState()
	if state != connectivity.Ready {
		return &plugin.HealthCheckResult{
			Status:    plugin.StatusDown,
			Message:   fmt.Sprintf("gRPC connection not ready, state: %s", state),
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
				"state":       state.String(),
			},
		}, nil
	}

	// If using the gRPC Health Checking Protocol
	if g.useHealth {
		healthClient := grpc_health_v1.NewHealthClient(conn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
			Service: g.service,
		})

		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Unimplemented {
				// The server doesn't implement the health checking protocol
				logger.Warn("gRPC server doesn't implement health checking protocol: %v", err)
				// Fall back to just checking the connection state
				return &plugin.HealthCheckResult{
					Status:    plugin.StatusUp,
					Message:   "gRPC connection established, but health check not implemented",
					Timestamp: time.Now().Unix(),
					Metadata: map[string]string{
						"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
						"state":       state.String(),
					},
				}, nil
			}

			return &plugin.HealthCheckResult{
				Status:    plugin.StatusDown,
				Message:   fmt.Sprintf("gRPC health check failed: %v", err),
				Timestamp: time.Now().Unix(),
				Metadata: map[string]string{
					"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
					"error":       err.Error(),
				},
			}, nil
		}

		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			return &plugin.HealthCheckResult{
				Status:    plugin.StatusDown,
				Message:   fmt.Sprintf("gRPC service not serving, status: %s", resp.Status),
				Timestamp: time.Now().Unix(),
				Metadata: map[string]string{
					"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
					"status":      resp.Status.String(),
				},
			}, nil
		}

		return &plugin.HealthCheckResult{
			Status:    plugin.StatusUp,
			Message:   "gRPC health check successful",
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
				"status":      resp.Status.String(),
			},
		}, nil
	}

	// If not using the health protocol, just check the connection state
	return &plugin.HealthCheckResult{
		Status:    plugin.StatusUp,
		Message:   "gRPC connection established",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
			"state":       state.String(),
		},
	}, nil
}
