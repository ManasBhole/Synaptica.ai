package routes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
)

type TrainingProxy struct {
	Client *http.Client
	Cfg    *config.Config
}

func NewTrainingProxy(client *http.Client, cfg *config.Config) *TrainingProxy {
	return &TrainingProxy{Client: client, Cfg: cfg}
}

func RegisterTrainingRoutes(router *mux.Router, proxy *TrainingProxy) {
	if proxy == nil || proxy.Client == nil || proxy.Cfg == nil {
		panic("training proxy requires client and config")
	}

	router.HandleFunc("/training/jobs", proxy.handleCreateJob).Methods(http.MethodPost)
	router.HandleFunc("/training/jobs", proxy.handleListJobs).Methods(http.MethodGet)
	router.HandleFunc("/training/jobs/{id}", proxy.handleGetJob).Methods(http.MethodGet)
	router.HandleFunc("/training/jobs/{id}/status", proxy.handleGetJob).Methods(http.MethodGet)
	router.HandleFunc("/training/jobs/{id}/artifact", proxy.handleArtifact).Methods(http.MethodGet)
}

func (p *TrainingProxy) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	p.forwardWithBody(w, r, http.MethodPost, fmt.Sprintf("%s/api/v1/training/jobs", p.Cfg.TrainingBaseURL))
}

func (p *TrainingProxy) handleListJobs(w http.ResponseWriter, r *http.Request) {
	endpoint := fmt.Sprintf("%s/api/v1/training/jobs", p.Cfg.TrainingBaseURL)
	p.forwardWithQuery(w, r, http.MethodGet, endpoint)
}

func (p *TrainingProxy) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	endpoint := fmt.Sprintf("%s/api/v1/training/jobs/%s", p.Cfg.TrainingBaseURL, id)
	p.forwardWithQuery(w, r, http.MethodGet, endpoint)
}

func (p *TrainingProxy) handleArtifact(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	endpoint := fmt.Sprintf("%s/api/v1/training/jobs/%s/artifact", p.Cfg.TrainingBaseURL, id)
	p.forwardWithQuery(w, r, http.MethodGet, endpoint)
}

func (p *TrainingProxy) forwardWithQuery(w http.ResponseWriter, r *http.Request, method, target string) {
	if len(r.URL.RawQuery) > 0 {
		target = fmt.Sprintf("%s?%s", target, r.URL.RawQuery)
	}
	p.forward(w, r, method, target, nil, false)
}

func (p *TrainingProxy) forwardWithBody(w http.ResponseWriter, r *http.Request, method, target string) {
	var body io.Reader
	if r.Body != nil {
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		body = bytes.NewReader(buf.Bytes())
		r.Body = io.NopCloser(buf)
	}
	p.forward(w, r, method, target, body, true)
}

func (p *TrainingProxy) forward(w http.ResponseWriter, r *http.Request, method, target string, body io.Reader, propagateBody bool) {
	ctx, cancel := context.WithTimeout(r.Context(), p.Cfg.GatewayRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}

	copyHeaders(r, req, propagateBody)

	corrID := ensureCorrelationID(req)

	resp, err := p.Client.Do(req)
	if err != nil {
		logger.Log.WithError(err).Error("training proxy failed")
		http.Error(w, "training service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, value := range v {
			w.Header().Add(k, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Log.WithError(err).Error("failed to copy training response")
	}

	logger.Log.WithFields(map[string]interface{}{
		"url":        target,
		"status":     resp.StatusCode,
		"request_id": corrID,
	}).Info("Forwarded request to training service")
}

func copyHeaders(src *http.Request, dst *http.Request, hasBody bool) {
	dst.Header = make(http.Header)
	for k, v := range src.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		dst.Header[k] = append([]string(nil), v...)
	}
	if hasBody {
		if ctype := src.Header.Get("Content-Type"); ctype != "" {
			dst.Header.Set("Content-Type", ctype)
		} else {
			dst.Header.Set("Content-Type", "application/json")
		}
	}
}

func ensureCorrelationID(req *http.Request) string {
	corrID := req.Header.Get("X-Request-ID")
	if corrID == "" {
		corrID = uuid.New().String()
		req.Header.Set("X-Request-ID", corrID)
	}
	return corrID
}
