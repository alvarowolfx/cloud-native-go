module github.com/alvarowolfx/cloud-native-go

go 1.15

require (
	github.com/apex/log v1.9.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/joho/godotenv v1.3.0
	github.com/kr/text v0.2.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.26.1
	go.opentelemetry.io/contrib/instrumentation/runtime v0.26.1
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/jaeger v1.1.0
	go.opentelemetry.io/otel/exporters/prometheus v0.24.0
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
	gocloud.dev v0.24.0
	gocloud.dev/docstore/mongodocstore v0.24.0
	gocloud.dev/pubsub/natspubsub v0.24.0
)
