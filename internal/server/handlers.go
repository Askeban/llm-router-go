package server

import (
	"encoding/json"
	"log"
	"net/http"

	"llm-router-go/internal/config"
	"llm-router-go/internal/inference"
	"llm-router-go/internal/selection"
	"llm-router-go/internal/types"
)

// SelectRequest represents the JSON payload for selecting a model.
type SelectRequest struct {
	Prompt      string                 `json:"prompt"`
	Context     *types.Context         `json:"context"`
	Preferences map[string]interface{} `json:"preferences"`
}

// SelectHandler returns an http.HandlerFunc that selects a model using the
// provided ConfigStore.  The handler reads the JSON payload, invokes the
// selection logic and writes the response as JSON.
func SelectHandler(store *config.ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode request
		var req SelectRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.Prompt == "" {
			http.Error(w, "prompt is required", http.StatusBadRequest)
			return
		}
		models := store.GetModels()
		result, err := selection.SelectModel(req.Prompt, req.Context, models, req.Preferences)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}
}

// GenerateResponse contains both the selected model details and the model's
// output text.
type GenerateResponse struct {
	Model  selection.Result `json:"model"`
	Output string           `json:"output"`
}

// GenerateHandler selects the best model for the given prompt and then forwards
// the prompt to that model, returning the model's output.
func GenerateHandler(store *config.ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SelectRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.Prompt == "" {
			http.Error(w, "prompt is required", http.StatusBadRequest)
			return
		}
		models := store.GetModels()
		result, err := selection.SelectModel(req.Prompt, req.Context, models, req.Preferences)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		output, err := inference.SendPrompt(result.RecommendedModel, req.Prompt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp := GenerateResponse{Model: result, Output: output}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}
}

// ModelsHandler writes the current models registry as JSON.
func ModelsHandler(store *config.ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models := store.GetModels()
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(models); err != nil {
			log.Printf("failed to write models response: %v", err)
		}
	}
}

// HealthHandler is a simple liveness check.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
