package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alvarowolfx/cloud-native-go/api"
	"github.com/alvarowolfx/cloud-native-go/cloud"
	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/apex/log"
	"github.com/joho/godotenv"
	"gocloud.dev/blob"

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
	serviceName := "api-server"
	telemetry.InitLogger()
	_ = telemetry.InitTracing(serviceName)
	_ = telemetry.InitMetrics(serviceName)
	log.Info("hello world")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sigs := make(chan os.Signal, 1)
	errs := make(chan error, 1)
	done := make(chan bool, 1)

	bucket, err := NewBucket("doc-files")
	if err != nil {
		log.Fatalf("failed to open bucket: %v", err)
	}
	defer bucket.Close()

	topic, err := cloud.NewTopic()
	if err != nil {
		log.Fatalf("failed to open pubsub topic: %v", err)
	}
	defer topic.Shutdown(context.Background())

	srv := api.NewServer(topic, bucket, port, errs)
	go srv.Start()

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
