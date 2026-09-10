package api

import (
	"encoding/json"
	"net/http"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// getConfig GET /configure (verify_auth; 敏感键 redact, T7)。
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	cfg := s.state.CurrentConfig()
	writeJSON(w, http.StatusOK, config.Redact(cfg, ""))
}

// listProviders GET /configure/providers (镜像内置集常量)。
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	writeJSON(w, http.StatusOK, map[string]any{
		"llm":      []string{"openai", "anthropic", "gemini"},
		"embedder": []string{"openai", "gemini"},
	})
}

// setConfig POST /configure (admin; 白名单 + 深合并 + 重建 + 持久化)。
func (s *Server) setConfig(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	updates, err := readAllJSON(r)
	if err != nil {
		write422(w, []pyErr{{Type: "model_attributes_type", Loc: []any{"body"},
			Msg: "Input should be a valid dictionary or object to extract fields from", Input: nil}})
		return
	}
	if err := validateBundledProviders(updates); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	persist := func(merged map[string]any) error {
		raw, err := json.Marshal(merged)
		if err != nil {
			return err
		}
		return s.store.UpsertSetting(r.Context(), "config_overrides", string(raw))
	}
	if _, err := s.state.UpdateConfig(updates, persist); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Configuration set successfully"})
}

// validateBundledProviders 对齐 _validate_bundled_providers (文案逐字, golden 锁定)。
func validateBundledProviders(cfg map[string]any) error {
	if llm, ok := cfg["llm"].(map[string]any); ok {
		if provider, ok := llm["provider"].(string); ok && provider != "" {
			if !contains(config.BundledLLMProviders, provider) {
				return errStr("LLM provider '" + provider + "' is not bundled in this image. Bundled providers: openai, anthropic, gemini. To use another provider, install its Python package, rebuild the container, and extend BUNDLED_LLM_PROVIDERS in server/main.py.")
			}
		}
	}
	if emb, ok := cfg["embedder"].(map[string]any); ok {
		if provider, ok := emb["provider"].(string); ok && provider != "" {
			if !contains(config.BundledEmbedderProviders, provider) {
				return errStr("Embedder provider '" + provider + "' is not bundled in this image. Bundled providers: openai, gemini. To use another provider, install its Python package, rebuild the container, and extend BUNDLED_EMBEDDER_PROVIDERS in server/main.py.")
			}
		}
	}
	return nil
}

// generateInstructions POST /generate-instructions (解析回退文案 golden 锁定)。
func (s *Server) generateInstructions(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	useCase := validateStringField(&errs, fields, map[string]any{}, "use_case", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	prompt := "You are configuring a memory system. Given the use case below, produce two things:\n" +
		"1. INSTRUCTIONS: A short paragraph of custom instructions telling the memory extraction system " +
		"what kinds of facts, preferences, and context to prioritize. Be specific to the use case.\n" +
		"2. TEST_MESSAGE: A single realistic sentence a user in this use case would say, suitable for " +
		"testing that the memory system works.\n\n" +
		"Respond in exactly this format (no markdown, no extra text):\n" +
		"INSTRUCTIONS: <your instructions>\n" +
		"TEST_MESSAGE: <your test message>\n\nUse case: " + *useCase
	resp, err := s.state.Memory().LLM.GenerateResponse(
		[]llmMessage{{Role: "user", Content: prompt}}, genOptsNone())
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	instructions := resp
	testMessage := "I like to hike on weekends."
	if containsFold(resp, "INSTRUCTIONS:") && containsFold(resp, "TEST_MESSAGE:") {
		parts := splitOnce(resp, "TEST_MESSAGE:")
		instructions = trimAll(replaceAll(parts[0], "INSTRUCTIONS:", ""))
		testMessage = trimAll(parts[1])
	}
	writeJSON(w, http.StatusOK, map[string]any{"custom_instructions": instructions, "test_message": testMessage})
}

// reset POST /reset (admin)。
func (s *Server) reset(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	if err := s.state.Memory().Reset(); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "All memories reset"})
}
