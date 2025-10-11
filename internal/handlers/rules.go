package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Gmacem/wasmorph/internal/wasm"
	"github.com/go-chi/chi/v5"
)

type RulesHandler struct {
	wasmService *wasm.Service
}

func NewRulesHandler(wasmService *wasm.Service) *RulesHandler {
	return &RulesHandler{
		wasmService: wasmService,
	}
}

func (h *RulesHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	userID := r.Header.Get("X-User-ID")
	rule, err := h.wasmService.SaveRule(r.Context(), userID, req.Name, req.Code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if rule.CreatedAt == rule.UpdatedAt {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Rule created"})
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Rule updated"})
	}
}

func (h *RulesHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	rules, err := h.wasmService.ListRules(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *RulesHandler) ExecuteRule(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	userID := r.Header.Get("X-User-ID")

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	result, err := h.wasmService.ExecuteRule(r.Context(), userID, name, input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var resultData any
	if err := json.Unmarshal(result, &resultData); err != nil {
		resultData = string(result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"result": resultData})
}

func (h *RulesHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	userID := r.Header.Get("X-User-ID")
	rule, err := h.wasmService.GetRule(r.Context(), userID, name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

func (h *RulesHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	userID := r.Header.Get("X-User-ID")

	if err := h.wasmService.DeleteRule(r.Context(), userID, name); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Rule deleted"})
}
