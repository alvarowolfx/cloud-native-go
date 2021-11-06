package api

import (
	"fmt"
	"net/http"
)

func (s *apiServer) handleQueryDocs(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.WithField("path", "/api/docs")
	logger.Infof("request received")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, "not allowed")
		return
	}
	fmt.Fprintln(w, "Hello, World!")
}
