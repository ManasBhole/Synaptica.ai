package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type LLMService struct {
	apiKey    string
	baseURL   string
	modelName string
}

func main() {
	logger.Init()
	cfg := config.Load()

	service := &LLMService{
		apiKey:    cfg.LLMAPIKey,
		baseURL:   cfg.LLMBaseURL,
		modelName: cfg.LLMModelName,
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/nl-to-cohort", service.handleNLToCohort).Methods("POST")
	router.HandleFunc("/api/v1/notes-nlp", service.handleNotesNLP).Methods("POST")
	router.HandleFunc("/api/v1/code-map", service.handleCodeMap).Methods("POST")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8086"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8086",
		}).Info("LLM Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down LLM Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("LLM Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *LLMService) handleNLToCohort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NaturalLanguage string `json:"natural_language"`
		Schema          map[string]interface{} `json:"schema"`
		Context         map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Convert natural language to cohort query
	cohortQuery, err := s.nlToCohort(req.NaturalLanguage, req.Schema, req.Context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cohortQuery)
}

func (s *LLMService) handleNotesNLP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Notes string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Extract structured data from notes using NLP
	result, err := s.processNotes(req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *LLMService) handleCodeMap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code        string `json:"code"`
		SourceSystem string `json:"source_system"`
		TargetSystems []string `json:"target_systems"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Map codes between systems (SNOMED, LOINC, ICD)
	mappings, err := s.mapCodes(req.Code, req.SourceSystem, req.TargetSystems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mappings)
}

func (s *LLMService) nlToCohort(nl string, schema map[string]interface{}, context map[string]interface{}) (models.CohortQuery, error) {
	// Use LLM to convert natural language to cohort query DSL
	prompt := fmt.Sprintf(`Convert the following natural language query to a cohort query DSL:
	
Natural Language: %s
Schema: %v
Context: %v

Return a JSON object with "dsl" and "description" fields.`, nl, schema, context)

	result, err := s.callLLM(prompt)
	if err != nil {
		return models.CohortQuery{}, err
	}

	// Parse LLM response
	var cohortQuery models.CohortQuery
	if err := json.Unmarshal([]byte(result), &cohortQuery); err != nil {
		// Fallback to simple parsing
		cohortQuery = models.CohortQuery{
			DSL:         result,
			Description: nl,
		}
	}

	return cohortQuery, nil
}

func (s *LLMService) processNotes(notes string) (map[string]interface{}, error) {
	// Use LLM to extract structured data from clinical notes
	prompt := fmt.Sprintf(`Extract structured medical information from the following clinical notes:
	
%s

Return a JSON object with extracted fields like: conditions, medications, procedures, observations.`, notes)

	result, err := s.callLLM(prompt)
	if err != nil {
		return nil, err
	}

	var structured map[string]interface{}
	if err := json.Unmarshal([]byte(result), &structured); err != nil {
		return map[string]interface{}{"raw": notes}, nil
	}

	return structured, nil
}

func (s *LLMService) mapCodes(code, sourceSystem string, targetSystems []string) (map[string]string, error) {
	// Use LLM to map codes between systems
	prompt := fmt.Sprintf(`Map the following medical code:
	
Code: %s
Source System: %s
Target Systems: %v

Return a JSON object with mappings for each target system.`, code, sourceSystem, targetSystems)

	result, err := s.callLLM(prompt)
	if err != nil {
		return nil, err
	}

	var mappings map[string]string
	if err := json.Unmarshal([]byte(result), &mappings); err != nil {
		return map[string]string{}, nil
	}

	return mappings, nil
}

func (s *LLMService) callLLM(prompt string) (string, error) {
	if s.apiKey == "" {
		// Mock response for development
		return `{"dsl": "SELECT * FROM patients WHERE age > 50", "description": "Patients over 50"}`, nil
	}

	// Call LLM API (OpenAI, Anthropic, etc.)
	payload := map[string]interface{}{
		"model": s.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from LLM")
}

