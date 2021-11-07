package worker

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/apex/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"gocloud.dev/blob"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
	"gocloud.dev/server"
	"gocloud.dev/server/health"
)

type worker struct {
	port string
	errs chan error

	logger *log.Entry
	coll   *docstore.Collection
	bucket *blob.Bucket
	sub    *pubsub.Subscription

	totalFilesProcessed metric.Int64Counter
	totalLinesProcessed metric.Int64Counter
	totalLinesWithError metric.Int64Counter
}

type Worker interface {
	Start()
}

func New(port string, errs chan error, coll *docstore.Collection, bucket *blob.Bucket, sub *pubsub.Subscription) Worker {
	logger := log.WithField("module", "worker")
	meter := global.GetMeterProvider().Meter("github.com/alvarowolfx/cloud-native-go")
	totalFilesProcessed, err := meter.NewInt64Counter("worker.files_processed.total", metric.WithDescription("total files processed"))
	handleOtelErr(err)
	totalLinesProcessed, err := meter.NewInt64Counter("worker.lines_processed.total", metric.WithDescription("total lines processed"))
	handleOtelErr(err)
	totalLinesWithError, err := meter.NewInt64Counter("worker.parse_errors.total", metric.WithDescription("total lines with error found"))
	handleOtelErr(err)
	return &worker{
		port:                port,
		errs:                errs,
		logger:              logger,
		coll:                coll,
		bucket:              bucket,
		sub:                 sub,
		totalFilesProcessed: totalFilesProcessed,
		totalLinesProcessed: totalLinesProcessed,
		totalLinesWithError: totalLinesWithError,
	}
}

func handleOtelErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}

func (w *worker) CheckHealth() error {
	ctx := context.Background()
	if ok, err := w.bucket.IsAccessible(ctx); !ok {
		e := fmt.Errorf("bucket is not accessible: %v", err)
		w.logger.Error(e.Error())
		return e
	}
	return nil
}

func (w *worker) Start() {
	go w.listenMessages()
	srvOptions := &server.Options{
		HealthChecks: []health.Checker{w},
	}
	srv := server.New(http.DefaultServeMux, srvOptions)

	w.logger.Infof("listening on port %s", w.port)
	err := srv.ListenAndServe(":" + w.port)
	if err != nil {
		w.errs <- err
	}
}

func (w *worker) downloadAndParse(ctx context.Context, jobId string) ([]map[string]interface{}, error) {
	r, err := w.bucket.NewReader(ctx, jobId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	csvReader := csv.NewReader(r)
	csvReader.LazyQuotes = true

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}
	records := make([]map[string]interface{}, 0)
	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			w.totalLinesWithError.Add(ctx, 1)
			w.logger.Errorf("failed to read csv: %v", err)
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
		w.totalLinesProcessed.Add(ctx, 1)
	}
	err = r.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close file: %v", err)
	}
	return records, nil
}

func (w *worker) listenMessages() {
	for {
		ctx := context.Background()
		msg, err := w.sub.Receive(ctx)
		if err != nil {
			log.Errorf("failed to receive message: %v", err)
			continue
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, telemetry.PubsubMetadataCarrier(msg.Metadata))

		tracer := otel.Tracer("worker")
		ctx, span := tracer.Start(ctx, "processing")

		jobId := string(msg.Body)
		w.logger.Infof("received message: %s - %v - %s", jobId, msg.Metadata, span.SpanContext().TraceID().String())

		ctx, spanDownloadFile := tracer.Start(ctx, "file.download")
		records, err := w.downloadAndParse(ctx, jobId)
		if err != nil {
			w.logger.Errorf("failed to download and parse file: %v", err)
			continue
		}
		spanDownloadFile.End()
		w.totalFilesProcessed.Add(ctx, 1)
		span.AddEvent("file.read")

		ctx, spanInsert := tracer.Start(ctx, "db.insert")
		actionList := w.coll.Actions()
		for _, record := range records {
			actionList.Create(record)
		}
		if err := actionList.Do(ctx); err != nil {
			w.logger.Errorf("failed to save records: %v", err)
			continue
		}
		spanInsert.End()

		msg.Ack()
		span.AddEvent("acked")
		span.End()
	}
}
