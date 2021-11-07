package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"gocloud.dev/docstore"
)

func (s *apiServer) handleQueryDocs(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.WithField("path", r.URL.Path)
	logger.Infof("request received")
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "not allowed")
		return
	}

	ctx := r.Context()
	iter := s.coll.Query().Get(ctx)
	defer iter.Stop()

	records, err := readDocuments(ctx, iter)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"list": records,
	})
}

func (s *apiServer) handleQueryByJobDocs(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.WithField("path", r.URL.Path)
	logger.Infof("request received")
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "not allowed")
		return
	}
	vars := mux.Vars(r)
	jobId := vars["jobId"]

	ctx := r.Context()
	iter := s.coll.Query().Where("jobId", "=", jobId).Get(ctx)
	defer iter.Stop()

	records, err := readDocuments(ctx, iter)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"list": records,
	})
}

func readDocuments(ctx context.Context, iter *docstore.DocumentIterator) ([]map[string]interface{}, error) {
	records := []map[string]interface{}{}
	for {
		r := map[string]interface{}{}
		err := iter.Next(ctx, r)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read query: %v", err)
		} else {
			records = append(records, r)
		}
	}
	return records, nil
}
