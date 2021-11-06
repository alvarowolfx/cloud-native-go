package main

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alvarowolfx/cloud-native-go/cloud"
	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/apex/log"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
	"gocloud.dev/blob"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
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
	go listenMessages(coll, bucket, sub)

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

func listenMessages(coll *docstore.Collection, bucket *blob.Bucket, sub *pubsub.Subscription) {
	logger := log.WithField("module", "worker")
	for {
		ctx := context.Background()
		msg, err := sub.Receive(ctx)
		if err != nil {
			log.Errorf("failed to receive message: %v", err)
			continue
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, telemetry.PubsubMetadataCarrier(msg.Metadata))

		tracer := otel.Tracer("worker")
		ctx, span := tracer.Start(ctx, "processing")
		jobId := string(msg.Body)
		logger.Infof("received message: %s - %v - %s", jobId, msg.Metadata, span.SpanContext().TraceID().String())

		ctx, spanDownloadFile := tracer.Start(ctx, "file.download")
		r, err := bucket.NewReader(ctx, string(msg.Body), nil)
		if err != nil {
			logger.Errorf("failed to read file: %v", err)
			continue
		}
		csvReader := csv.NewReader(r)
		csvReader.LazyQuotes = true

		header, err := csvReader.Read()
		if err != nil {
			logger.Errorf("failed to read header: %v", err)
			continue
		}
		records := make([]map[string]interface{}, 0)
		for {
			line, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Errorf("failed to read csv: %v", err)
				continue
			}
			record := map[string]interface{}{
				"jobId": jobId,
			}
			for i, v := range line {
				h := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(header[i], "\"", "")))
				cv := strings.TrimSpace(strings.ReplaceAll(v, "\"", ""))
				record[h] = cv
			}
			records = append(records, record)
		}
		spanDownloadFile.End()
		span.AddEvent("file.read")

		err = r.Close()
		if err != nil {
			logger.Errorf("failed to close file: %v", err)
			continue
		}

		ctx, spanInsert := tracer.Start(ctx, "db.insert")
		actionList := coll.Actions()
		for _, record := range records {
			actionList.Create(record)
		}
		if err := actionList.Do(ctx); err != nil {
			logger.Errorf("failed to save records: %v", err)
			continue
		}
		spanInsert.End()

		msg.Ack()
		span.AddEvent("acked")
		span.End()
	}
}
