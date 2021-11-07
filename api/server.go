package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"gocloud.dev/blob"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
	"gocloud.dev/server"
	"gocloud.dev/server/health"
)

type Server interface {
	Start()
}

type apiServer struct {
	port   string
	errs   chan error
	logger *log.Entry

	bucket *blob.Bucket
	topic  *pubsub.Topic
	coll   *docstore.Collection

	totalFileUploaded     metric.Int64Counter
	totalFileSizeUploaded metric.Int64Counter
}

func NewServer(coll *docstore.Collection, topic *pubsub.Topic, bucket *blob.Bucket, port string, errs chan error) Server {
	logger := log.WithField("module", "api")

	meter := global.GetMeterProvider().Meter("github.com/alvarowolfx/cloud-native-go")
	totalFileUploaded, err := meter.NewInt64Counter("api.file_upload.total", metric.WithDescription("total number file uploaded"))
	handleOtelErr(err)
	totalFileSizeUploaded, err := meter.NewInt64Counter("api.file.upload.size", metric.WithDescription("total size of file uploaded"))
	handleOtelErr(err)

	return &apiServer{
		port:                  port,
		errs:                  errs,
		logger:                logger,
		coll:                  coll,
		topic:                 topic,
		bucket:                bucket,
		totalFileUploaded:     totalFileUploaded,
		totalFileSizeUploaded: totalFileSizeUploaded,
	}
}

func handleOtelErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}

func (s *apiServer) CheckHealth() error {
	ctx := context.Background()
	if ok, err := s.bucket.IsAccessible(ctx); !ok {
		e := fmt.Errorf("bucket is not accessible: %v", err)
		s.logger.Error(e.Error())
		return e
	}
	return nil
}

func (s *apiServer) sendError(w http.ResponseWriter, statusCode int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": errorMsg})
}

func (s *apiServer) traceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tracer := otel.Tracer("api")
		_, span := tracer.Start(ctx, "request")
		defer span.End()
		logger := s.logger.
			WithField("spanId", span.SpanContext().SpanID().String()).
			WithField("traceId", span.SpanContext().TraceID().String())
		logger.Info("trace request")
		next.ServeHTTP(w, r)
	})
}

func (s *apiServer) Start() {
	srvOptions := &server.Options{
		HealthChecks: []health.Checker{s},
	}
	srv := server.New(http.DefaultServeMux, srvOptions)

	r := mux.NewRouter()
	r.Use(s.traceMiddleware)
	r.HandleFunc("/api/docs/upload", s.handleDocsUpload)
	r.HandleFunc("/api/{jobId}/docs", s.handleQueryByJobDocs)
	r.HandleFunc("/api/docs", s.handleQueryDocs)

	http.Handle("/", otelhttp.NewHandler(r, "api"))

	s.logger.Infof("listening on port %s", s.port)
	err := srv.ListenAndServe(":" + s.port)
	if err != nil {
		s.errs <- err
	}
}
