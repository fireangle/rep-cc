package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	},
}

// streamRawSSE pipes upstream SSE bytes directly to client with zero parsing.
// Used when upstream and client use the same format.
func streamRawSSE(w http.ResponseWriter, resp *http.Response) {
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		if line == "" {
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

// doUpstreamRequest sends a JSON body to upstream and returns the response.
func doUpstreamRequest(method, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient.Do(req)
}

func writeSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": msg,
			"type":    "error",
		},
	})
}

// --- /v1/models ---

func handleModels(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	resp := ModelListResponse{
		Object: "list",
		Data:   make([]ModelData, len(allModels)),
	}
	for i, m := range allModels {
		resp.Data[i] = ModelData{
			ID:      m.ID,
			Object:  "model",
			Created: now,
			OwnedBy: m.Provider,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- /v1/chat/completions ---

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, 400, "failed to read request body")
		return
	}

	var req OpenAIChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, 400, "invalid JSON")
		return
	}

	if req.Model == "" {
		writeJSONError(w, 400, "model is required")
		return
	}

	if isOpenAIModel(req.Model) {
		handleChatCompletionsOpenAI(w, r, body, &req)
	} else if isAnthropicModel(req.Model) {
		handleChatCompletionsAnthropic(w, r, &req)
	} else {
		writeJSONError(w, 400, fmt.Sprintf("Unknown model: %s", req.Model))
	}
}

func handleChatCompletionsOpenAI(w http.ResponseWriter, r *http.Request, rawBody []byte, req *OpenAIChatRequest) {
	url := cfg.OpenAIBaseURL + "/chat/completions"
	headers := map[string]string{
		"Authorization": "Bearer " + cfg.OpenAIAPIKey,
	}

	resp, err := doUpstreamRequest("POST", url, rawBody, headers)
	if err != nil {
		writeJSONError(w, 502, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if req.Stream {
		// Same format → zero-copy pipe
		writeSSEHeaders(w)
		w.WriteHeader(resp.StatusCode)
		streamRawSSE(w, resp)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func handleChatCompletionsAnthropic(w http.ResponseWriter, r *http.Request, req *OpenAIChatRequest) {
	anthropicReq := buildAnthropicRequest(req)

	if req.Stream {
		anthropicReq.Stream = true
		body, _ := json.Marshal(anthropicReq)

		url := cfg.AnthropicBaseURL + "/v1/messages"
		headers := map[string]string{
			"x-api-key":         cfg.AnthropicAPIKey,
			"anthropic-version": "2023-06-01",
		}

		resp, err := doUpstreamRequest("POST", url, body, headers)
		if err != nil {
			writeJSONError(w, 502, "upstream request failed: "+err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}

		writeSSEHeaders(w)
		flusher, _ := w.(http.Flusher)

		// Translate Anthropic SSE → OpenAI SSE
		var msgID string
		var inputTokens, outputTokens int

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var event AnthropicSSEEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil {
					msgID = event.Message.ID
					inputTokens = event.Message.Usage.InputTokens
				}

			case "content_block_delta":
				var delta AnthropicDelta
				if json.Unmarshal(event.Delta, &delta) != nil {
					continue
				}
				if delta.Type == "text_delta" && delta.Text != "" {
					chunk := OpenAIChatCompletion{
						ID:      msgID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []OpenAIChoice{
							{
								Index: 0,
								Delta: &OpenAIRespMsg{
									Role:    "assistant",
									Content: &delta.Text,
								},
							},
						},
					}
					b, _ := json.Marshal(chunk)
					fmt.Fprintf(w, "data: %s\n\n", b)
					if flusher != nil {
						flusher.Flush()
					}
				}

			case "message_delta":
				var md struct {
					Usage struct {
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
					StopReason *string `json:"stop_reason"`
				}
				if json.Unmarshal(event.Delta, &md) == nil {
					outputTokens = md.Usage.OutputTokens
				}

			case "message_stop":
				// Send final chunk
				finishReason := "stop"
				chunk := OpenAIChatCompletion{
					ID:      msgID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []OpenAIChoice{
						{
							Index:        0,
							Delta:        &OpenAIRespMsg{},
							FinishReason: &finishReason,
						},
					},
					Usage: &OpenAIUsage{
						PromptTokens:     inputTokens,
						CompletionTokens: outputTokens,
						TotalTokens:      inputTokens + outputTokens,
					},
				}
				b, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", b)
				if flusher != nil {
					flusher.Flush()
				}
			}
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	} else {
		// Non-streaming
		body, _ := json.Marshal(anthropicReq)
		url := cfg.AnthropicBaseURL + "/v1/messages"
		headers := map[string]string{
			"x-api-key":         cfg.AnthropicAPIKey,
			"anthropic-version": "2023-06-01",
		}

		resp, err := doUpstreamRequest("POST", url, body, headers)
		if err != nil {
			writeJSONError(w, 502, "upstream request failed: "+err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}

		var anthropicResp AnthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			writeJSONError(w, 502, "failed to decode upstream response")
			return
		}

		openAIResp := anthropicResponseToOpenAI(&anthropicResp)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResp)
	}
}

// --- /v1/messages ---

func handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, 400, "failed to read request body")
		return
	}

	var peek struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &peek); err != nil {
		writeJSONError(w, 400, "invalid JSON")
		return
	}

	if peek.Model == "" {
		writeJSONError(w, 400, "model is required")
		return
	}

	if isAnthropicModel(peek.Model) {
		handleMessagesAnthropic(w, r, body, peek.Stream)
	} else if isOpenAIModel(peek.Model) {
		handleMessagesOpenAI(w, r, body, peek.Stream)
	} else {
		writeJSONError(w, 400, fmt.Sprintf("Unknown model: %s", peek.Model))
	}
}

func handleMessagesAnthropic(w http.ResponseWriter, r *http.Request, rawBody []byte, stream bool) {
	// Sanitize: remove unsupported fields
	rawBody = sanitizeAnthropicBody(rawBody)

	url := cfg.AnthropicBaseURL + "/v1/messages"
	headers := map[string]string{
		"x-api-key":         cfg.AnthropicAPIKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := doUpstreamRequest("POST", url, rawBody, headers)
	if err != nil {
		writeJSONError(w, 502, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if stream {
		// Native Anthropic SSE → zero-copy pipe
		writeSSEHeaders(w)
		w.WriteHeader(resp.StatusCode)
		streamRawSSE(w, resp)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func handleMessagesOpenAI(w http.ResponseWriter, r *http.Request, rawBody []byte, stream bool) {
	url := cfg.OpenAIBaseURL + "/chat/completions"
	headers := map[string]string{
		"Authorization": "Bearer " + cfg.OpenAIAPIKey,
	}

	// Rewrite stream flag into body
	var bodyMap map[string]interface{}
	json.Unmarshal(rawBody, &bodyMap)
	if stream {
		bodyMap["stream"] = true
	}
	rawBody, _ = json.Marshal(bodyMap)

	resp, err := doUpstreamRequest("POST", url, rawBody, headers)
	if err != nil {
		writeJSONError(w, 502, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if stream {
		writeSSEHeaders(w)
		w.WriteHeader(resp.StatusCode)
		streamRawSSE(w, resp)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

// sanitizeAnthropicBody removes unsupported fields like output_config
func sanitizeAnthropicBody(body []byte) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil {
		return body
	}

	changed := false
	for _, field := range []string{"output_config"} {
		if _, ok := m[field]; ok {
			delete(m, field)
			changed = true
		}
	}

	// Ensure max_tokens has default
	if _, ok := m["max_tokens"]; !ok {
		m["max_tokens"] = json.RawMessage(`8192`)
		changed = true
	}

	if !changed {
		return body
	}

	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	}
}
