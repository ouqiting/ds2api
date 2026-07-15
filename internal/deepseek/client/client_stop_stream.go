package client

import (
	"bufio"
	"context"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/config"
)

func (c *Client) StopStream(ctx context.Context, a *auth.RequestAuth, sessionID string, messageID int) error {
	if strings.TrimSpace(sessionID) == "" || messageID <= 0 {
		return errors.New("missing stop_stream identifiers")
	}
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	payload := map[string]any{
		"chat_session_id": sessionID,
		"message_id":      messageID,
	}
	resp, status, err := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekStopStreamURL, headers, payload)
	if err != nil {
		config.Logger.Warn("[stop_stream] request error", "session_id", sessionID, "message_id", messageID, "account", a.AccountID, "error", err)
		return err
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(resp)
	if status != http.StatusOK || code != 0 || bizCode != 0 {
		config.Logger.Warn("[stop_stream] non-success response", "session_id", sessionID, "message_id", messageID, "account", a.AccountID, "status", status, "code", code, "biz_code", bizCode, "msg", msg, "biz_msg", bizMsg, "resp", fmt.Sprintf("%v", resp))
		return fmt.Errorf("stop_stream failed: status=%d code=%d biz_code=%d msg=%s biz_msg=%s", status, code, bizCode, msg, bizMsg)
	}
	config.Logger.Info("[stop_stream] ok", "session_id", sessionID, "message_id", messageID, "account", a.AccountID)
	return nil
}

func (c *Client) FireCompletionAndStop(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, stopDelay time.Duration) (int, error) {
	sessionID, _ := payload["chat_session_id"].(string)
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	headers["x-ds-pow-response"] = powResp
	captureSession := c.capture.Start("deepseek_completion", dsprotocol.DeepSeekCompletionURL, a.AccountID, payload)
	resp, err := c.streamPostOnce(ctx, clients.stream, dsprotocol.DeepSeekCompletionURL, headers, payload)
	if err != nil {
		config.Logger.Warn("[fire_completion_and_stop] completion request failed", "session_id", sessionID, "account", a.AccountID, "error", err)
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			config.Logger.Warn("[fire_completion_and_stop] response body close failed", "error", err)
		}
	}()
	if captureSession != nil {
		resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		config.Logger.Warn("[fire_completion_and_stop] completion returned non-200", "session_id", sessionID, "account", a.AccountID, "status", resp.StatusCode)
		return 0, fmt.Errorf("completion returned HTTP %d", resp.StatusCode)
	}

	newBody, muted, muteUntil, err := detectMutedCompletion(resp.Body)
	if err != nil {
		config.Logger.Warn("[fire_completion_and_stop] mute detection failed", "session_id", sessionID, "account", a.AccountID, "error", err)
		return 0, err
	}
	if muted {
		config.Logger.Warn("[fire_completion_and_stop] account muted", "session_id", sessionID, "account", a.AccountID)
		c.persistMutedUntil(a.AccountID, muteUntil)
		return 0, &RequestFailure{Op: "completion", Kind: FailureMuted, Message: "user is muted"}
	}
	if newBody != nil {
		resp.Body = newBody
	}

	responseMessageID := 0
	requestMessageID := 0
	initialStableSeen := false
	var ssePreview strings.Builder
	reader := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := strings.TrimSpace(string(line))
			if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
				if ssePreview.Len() < 2000 && trimmed != "" {
					if ssePreview.Len() > 0 {
						ssePreview.WriteByte('\n')
					}
					ssePreview.WriteString(trimmed)
				}
			} else {
				if ssePreview.Len() < 2000 {
					if ssePreview.Len() > 0 {
						ssePreview.WriteByte('\n')
					}
					ssePreview.WriteString(trimmed)
				}
				data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if data != "[DONE]" {
					var chunk map[string]any
					if json.Unmarshal([]byte(data), &chunk) == nil {
						if id := intFrom(chunk["request_message_id"]); id > 0 && requestMessageID == 0 {
							requestMessageID = id
						}
						extractResponseMessageID(chunk, &responseMessageID)
						if stoppedSegmentStableChunk(chunk) {
							initialStableSeen = true
						}
					}
				}
			}
			if responseMessageID > 0 {
				break
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			config.Logger.Warn("[fire_completion_and_stop] SSE scan error", "session_id", sessionID, "account", a.AccountID, "error", readErr)
			return 0, readErr
		}
	}
	if responseMessageID <= 0 {
		previewStr := ssePreview.String()
		if strings.HasPrefix(strings.TrimSpace(previewStr), "{") {
			var errResp map[string]any
			if json.Unmarshal([]byte(previewStr), &errResp) == nil {
				code := intFrom(errResp["code"])
				msg, _ := errResp["msg"].(string)
				if msg == "" {
					if data, _ := errResp["data"].(map[string]any); data != nil {
						msg, _ = data["biz_msg"].(string)
					}
				}
				if code != 0 || msg != "" {
					config.Logger.Warn("[fire_completion_and_stop] upstream JSON error", "session_id", sessionID, "account", a.AccountID, "parent_message_id", payload["parent_message_id"], "code", code, "msg", msg, "raw", previewStr)
					return 0, fmt.Errorf("completion upstream error: code=%d msg=%s", code, msg)
				}
			}
		}
		config.Logger.Warn("[fire_completion_and_stop] response_message_id not received before stream ended", "session_id", sessionID, "account", a.AccountID, "parent_message_id", payload["parent_message_id"], "request_message_id", requestMessageID, "sse_preview", previewStr)
		return 0, errors.New("response_message_id not received before stream ended")
	}
	config.Logger.Info("[fire_completion_and_stop] captured ids", "session_id", sessionID, "account", a.AccountID, "request_message_id", requestMessageID, "response_message_id", responseMessageID)

	stableCh := make(chan struct{}, 1)
	signalStable := func() {
		select {
		case stableCh <- struct{}{}:
		default:
		}
	}
	drainDone := make(chan struct{})
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				observeStoppedSegmentStableLine(line, signalStable)
			}
			if err != nil {
				if err != io.EOF {
					config.Logger.Warn("[fire_completion_and_stop] drain stream error", "session_id", sessionID, "account", a.AccountID, "error", err)
				}
				break
			}
		}
		close(drainDone)
	}()

	stableSeen := initialStableSeen
	drainClosed := false
	forceStop := false
	if stopDelay > 0 {
		stopTimer := time.NewTimer(stopDelay)
		stableFallbackDelay := stopDelay + 4*time.Second
		stableTimer := time.NewTimer(stableFallbackDelay)
		stopDue := false
		for !stopDue || (!stableSeen && !forceStop) {
			select {
			case <-stopTimer.C:
				stopDue = true
			case <-stableCh:
				stableSeen = true
			case <-drainDone:
				drainClosed = true
				if !stableSeen {
					config.Logger.Warn("[fire_completion_and_stop] stream ended before stable response event", "session_id", sessionID, "account", a.AccountID, "request_message_id", requestMessageID, "response_message_id", responseMessageID)
				}
				stopDue = true
				forceStop = true
			case <-stableTimer.C:
				if !stableSeen {
					config.Logger.Warn("[fire_completion_and_stop] stable response wait timed out before stop", "session_id", sessionID, "account", a.AccountID, "request_message_id", requestMessageID, "response_message_id", responseMessageID, "wait", stableFallbackDelay)
				}
				stopDue = true
				forceStop = true
			case <-ctx.Done():
				if !stopTimer.Stop() {
					select {
					case <-stopTimer.C:
					default:
					}
				}
				if !stableTimer.Stop() {
					select {
					case <-stableTimer.C:
					default:
					}
				}
				config.Logger.Warn("[fire_completion_and_stop] context cancelled during stop delay", "session_id", sessionID, "account", a.AccountID, "error", ctx.Err())
				return responseMessageID, ctx.Err()
			}
		}
		if !stopTimer.Stop() {
			select {
			case <-stopTimer.C:
			default:
			}
		}
		if !stableTimer.Stop() {
			select {
			case <-stableTimer.C:
			default:
			}
		}
	}

	stopCalledAt := time.Now()
	if err := c.StopStream(ctx, a, sessionID, responseMessageID); err != nil {
		config.Logger.Warn("[fire_completion_and_stop] stop_stream failed", "session_id", sessionID, "message_id", responseMessageID, "error", err)
		return 0, err
	}

	if !drainClosed {
		select {
		case <-drainDone:
		case <-time.After(10 * time.Second):
			config.Logger.Warn("[fire_completion_and_stop] drain stream timed out, forcing close", "session_id", sessionID, "account", a.AccountID)
		}
	}

	config.Logger.Info("[fire_completion_and_stop] segment sent and stopped", "session_id", sessionID, "response_message_id", responseMessageID, "request_message_id", requestMessageID, "stop_delay", stopDelay, "drain_after_stop", time.Since(stopCalledAt), "stable_seen", stableSeen, "account", a.AccountID)
	return responseMessageID, nil
}

func observeStoppedSegmentStableLine(line []byte, signalStable func()) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
		return
	}
	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if data == "" || data == "[DONE]" {
		return
	}
	var chunk map[string]any
	if json.Unmarshal([]byte(data), &chunk) != nil {
		return
	}
	if stoppedSegmentStableChunk(chunk) {
		signalStable()
	}
}

func stoppedSegmentStableChunk(chunk map[string]any) bool {
	if p, _ := chunk["p"].(string); strings.HasPrefix(p, "response/") {
		return true
	}
	if v, _ := chunk["v"].(map[string]any); v != nil {
		if response, _ := v["response"].(map[string]any); response != nil {
			return true
		}
	}
	if message, _ := chunk["message"].(map[string]any); message != nil {
		if response, _ := message["response"].(map[string]any); response != nil {
			return true
		}
	}
	return false
}

func extractResponseMessageID(chunk map[string]any, out *int) {
	if chunk == nil || out == nil {
		return
	}
	if id := intFrom(chunk["response_message_id"]); id > 0 {
		*out = id
	}
	if v, ok := chunk["v"].(map[string]any); ok {
		if response, _ := v["response"].(map[string]any); response != nil {
			if id := intFrom(response["message_id"]); id > 0 {
				*out = id
			}
		}
	}
	if message, _ := chunk["message"].(map[string]any); message != nil {
		if response, _ := message["response"].(map[string]any); response != nil {
			if id := intFrom(response["message_id"]); id > 0 {
				*out = id
			}
		}
	}
}
