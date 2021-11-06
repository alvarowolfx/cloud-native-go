package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alvarowolfx/cloud-native-go/cloud"
	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/alvarowolfx/cloud-native-go/worker"
	"github.com/apex/log"
	"github.com/joho/godotenv"
)

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

	bucket, err := cloud.NewBucket("doc-files")
	if err != nil {
		log.Fatalf("failed to open bucket: %v", err)
	}
	defer bucket.Close()

	coll, err := cloud.NewDocstore("docs", "id")
	if err != nil {
		log.Fatalf("failed to open collection: %v", err)
	}
	defer coll.Close()

	sub, err := cloud.NewTopicSub()
	if err != nil {
		log.Fatalf("failed to open pubsub topic: %v", err)
	}
	defer func() {
		err := sub.Shutdown(context.Background())
		if err != nil {
			log.Errorf("failed to shutdown pubsub topic: %v", err)
			errs <- err
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	w := worker.New(port, errs, coll, bucket, sub)
	go w.Start()

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
