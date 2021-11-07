package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alvarowolfx/cloud-native-go/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"gocloud.dev/pubsub"
)

func (s *apiServer) handleDocsUpload(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("api")
	ctx := r.Context()

	logger := s.logger.WithField("path", "/api/docs/upload")
	logger.Infof("request received")
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		errorMsg := fmt.Sprintf("missing file: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusBadRequest, errorMsg)
		return
	}
	defer file.Close()
	logger.WithField("size", handler.Size).WithField("filename", handler.Filename).Infof("file received")

	ctx, spanParse := tracer.Start(ctx, "csv.parse")
	defer spanParse.End()
	csvReader := csv.NewReader(file)
	csvReader.LazyQuotes = true
	_, err = csvReader.Read()
	if err != nil {
		errorMsg := fmt.Sprintf("failed to parse file: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusInternalServerError, errorMsg)
		return
	}
	spanParse.End()

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to read file: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusInternalServerError, errorMsg)
		return
	}

	ctx, spanUpload := tracer.Start(ctx, "file.upload")
	defer spanUpload.End()
	jobId := uuid.NewString()
	writer, err := s.bucket.NewWriter(ctx, jobId, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to save file: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusInternalServerError, errorMsg)
		return
	}

	totalRead, err := writer.ReadFrom(file)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to transfer file: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusInternalServerError, errorMsg)
		return
	}
	writer.Close()
	spanUpload.End()

	s.totalFileUploaded.Add(ctx, 1)
	s.totalFileSizeUploaded.Add(ctx, totalRead)

	msg := &pubsub.Message{
		Body: []byte(jobId),
		Metadata: map[string]string{
			"eventType": "file.upload",
		},
	}
	otel.GetTextMapPropagator().Inject(ctx, telemetry.PubsubMetadataCarrier(msg.Metadata))
	err = s.topic.Send(ctx, msg)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to queue file to be processed: %v", err)
		logger.Error(errorMsg)
		s.sendError(w, http.StatusInternalServerError, errorMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":        jobId,
		"totalRead": fmt.Sprintf("%v", totalRead),
		"size":      fmt.Sprintf("%v", handler.Size),
	})
}
