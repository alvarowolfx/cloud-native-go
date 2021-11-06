package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func GetResource(serviceName string) (*resource.Resource, error) {
	ctx := context.Background()
	return resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
}
