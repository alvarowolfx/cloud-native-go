package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alvarowolfx/cloud-native-go/cloud"
	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/apex/log"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
	"gocloud.dev/blob"
	"gocloud.dev/pubsub"

	// Import providers for blob storage
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"

	// Import providers for pubsub
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"
)

func NewBucket(prefix string) (*blob.Bucket, error) {
	ctx := context.Background()
	url := os.Getenv("BUCKET_URL")
	if url == "" {
		url = "file://./tmp/"
	}
	bucket, err := blob.OpenBucket(ctx, url+"?prefix="+prefix)
	if err != nil {
		return nil, fmt.Errorf("could not open bucket: %v", err)
	}
	return bucket, nil
}

func main() {
	_ = godotenv.Load()
	serviceName := "worker"
	telemetry.InitLogger()
	_ = telemetry.InitTracing(serviceName)
	_ = telemetry.InitMetrics(serviceName)
	log.Info("hello world")

	sigs := make(chan os.Signal, 1)
	errs := make(chan error, 1)
	done := make(chan bool, 1)

	bucket, err := NewBucket("doc-files")
	if err != nil {
		log.Fatalf("failed to open bucket: %v", err)
	}
	defer bucket.Close()

	sub, err := cloud.NewTopicSub()
	if err != nil {
		log.Fatalf("failed to open pubsub topic: %v", err)
	}
	defer sub.Shutdown(context.Background())
	go listenMessages(bucket, sub)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigs:
			log.Infof("received signal %s", sig)
			done <- true
		case err := <-errs:
			log.Errorf("error: %s", err)
			done <- true
		}
	}()

	log.Info("waiting shutdown")
	<-done
	log.Info("shutdown")
}

func listenMessages(bucket *blob.Bucket, sub *pubsub.Subscription) {
	logger := log.WithField("module", "worker")
	for {
		ctx := context.Background()
		msg, err := sub.Receive(ctx)
		if err != nil {
			log.Errorf("failed to receive message: %v", err)
			continue
		}
		carrier := telemetry.PubsubCarrier{Message: msg}
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

		tracer := otel.Tracer("worker")
		ctx, span := tracer.Start(ctx, "processing")
		logger.Infof("received message: %s - %v - %s", msg.Body, msg.Metadata, span.SpanContext().TraceID().String())

		ctx, spanDownloadFile := tracer.Start(ctx, "file.download")
		_, err = bucket.ReadAll(ctx, string(msg.Body))
		if err != nil {
			logger.Errorf("failed to read file: %v", err)
			continue
		}
		spanDownloadFile.End()
		span.AddEvent("file.read")

		span.AddEvent("acked")
		span.End()
		msg.Ack()
	}
}
