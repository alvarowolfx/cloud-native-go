package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alvarowolfx/cloud-native-go/api"
	"github.com/alvarowolfx/cloud-native-go/cloud"
	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/apex/log"
	"github.com/joho/godotenv"
)

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

	bucket, err := cloud.NewBucket("doc-files")
	if err != nil {
		log.Fatalf("failed to open bucket: %v", err)
	}
	defer bucket.Close()

	coll, err := cloud.NewDocstore("docs", "id")
	if err != nil {
		log.Fatalf("failed to open docstore: %v", err)
	}

	topic, err := cloud.NewTopic()
	if err != nil {
		log.Fatalf("failed to open pubsub topic: %v", err)
	}
	defer topic.Shutdown(context.Background())

	srv := api.NewServer(coll, topic, bucket, port, errs)
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
