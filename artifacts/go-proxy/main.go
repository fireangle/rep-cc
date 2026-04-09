package main

import (
        "log"
        "net/http"
        "os"
        "strings"
)

type Config struct {
        Port             string
        ProxyAPIKey      string
        OpenAIBaseURL    string
        OpenAIAPIKey     string
        AnthropicBaseURL string
        AnthropicAPIKey  string
}

var cfg Config

var allModels = []ModelInfo{
        {ID: "gpt-5.2", Provider: "openai"},
        {ID: "gpt-5-mini", Provider: "openai"},
        {ID: "gpt-5-nano", Provider: "openai"},
        {ID: "o4-mini", Provider: "openai"},
        {ID: "o3", Provider: "openai"},
        {ID: "claude-opus-4-6", Provider: "anthropic"},
        {ID: "claude-sonnet-4-6", Provider: "anthropic"},
        {ID: "claude-haiku-4-5", Provider: "anthropic"},
}

func isOpenAIModel(model string) bool {
        return strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o")
}

func isAnthropicModel(model string) bool {
        return strings.HasPrefix(model, "claude-")
}

func env(key, fallback string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return fallback
}

// authMiddleware checks Bearer token or x-api-key. Always enforces auth — fails closed if PROXY_API_KEY is unset.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                token := ""
                if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
                        token = auth[7:]
                }
                if token == "" {
                        token = r.Header.Get("x-api-key")
                }

                if token == "" || token != cfg.ProxyAPIKey {
                        writeJSONError(w, 401, "Unauthorized")
                        return
                }
                next(w, r)
        }
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key")

                if r.Method == http.MethodOptions {
                        w.WriteHeader(204)
                        return
                }
                next.ServeHTTP(w, r)
        })
}

func main() {
        cfg = Config{
                Port:             env("PORT", "8080"),
                ProxyAPIKey:      os.Getenv("PROXY_API_KEY"),
                OpenAIBaseURL:    env("AI_INTEGRATIONS_OPENAI_BASE_URL", "https://api.openai.com/v1"),
                OpenAIAPIKey:     env("AI_INTEGRATIONS_OPENAI_API_KEY", "dummy"),
                AnthropicBaseURL: env("AI_INTEGRATIONS_ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
                AnthropicAPIKey:  env("AI_INTEGRATIONS_ANTHROPIC_API_KEY", "dummy"),
        }

        // Strip trailing /v1 from Anthropic base URL since we add it in proxy
        cfg.AnthropicBaseURL = strings.TrimSuffix(cfg.AnthropicBaseURL, "/v1")
        cfg.AnthropicBaseURL = strings.TrimSuffix(cfg.AnthropicBaseURL, "/")

        // Strip trailing slash from OpenAI base URL (already includes /v1 typically)
        cfg.OpenAIBaseURL = strings.TrimSuffix(cfg.OpenAIBaseURL, "/")

        mux := http.NewServeMux()

        // Health check
        mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write([]byte(`{"status":"ok"}`))
        })

        // Proxy routes (with auth)
        mux.HandleFunc("GET /v1/models", logRequest(authMiddleware(handleModels)))
        mux.HandleFunc("POST /v1/chat/completions", logRequest(authMiddleware(handleChatCompletions)))
        mux.HandleFunc("POST /v1/messages", logRequest(authMiddleware(handleMessages)))

        handler := corsMiddleware(mux)

        if cfg.ProxyAPIKey == "" {
                log.Fatal("FATAL: PROXY_API_KEY is not set — refusing to start without authentication")
        }

        log.Printf("Go proxy listening on :%s", cfg.Port)
        if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
                log.Fatal(err)
        }
}
