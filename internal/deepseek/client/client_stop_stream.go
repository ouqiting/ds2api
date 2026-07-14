package client

import (
	"context"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"encoding/json"
	"errors"
	"fmt"
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
		return err
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(resp)
	if status != http.StatusOK || code != 0 || bizCode != 0 {
		return fmt.Errorf("stop_stream failed: status=%d code=%d biz_code=%d msg=%s biz_msg=%s", status, code, bizCode, msg, bizMsg)
	}
	config.Logger.Debug("[stop_stream] ok", "session_id", sessionID, "message_id", messageID, "account", a.AccountID)
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
		return 0, fmt.Errorf("completion returned HTTP %d", resp.StatusCode)
	}

	newBody, muted, muteUntil, err := detectMutedCompletion(resp.Body)
	if err != nil {
		return 0, err
	}
	if muted {
		c.persistMutedUntil(a.AccountID, muteUntil)
		return 0, &RequestFailure{Op: "completion", Kind: FailureMuted, Message: "user is muted"}
	}
	if newBody != nil {
		resp.Body = newBody
	}

	responseMessageID := 0
	scanErr := dsprotocol.ScanSSELines(resp, func(line []byte) bool {
		if responseMessageID > 0 {
			return false
		}
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
			return true
		}
		data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		if data == "[DONE]" {
			return false
		}
		var chunk map[string]any
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return true
		}
		extractResponseMessageID(chunk, &responseMessageID)
		return responseMessageID == 0
	})
	if scanErr != nil {
		return 0, scanErr
	}
	if responseMessageID <= 0 {
		return 0, errors.New("response_message_id not received before stream ended")
	}

	if stopDelay > 0 {
		select {
		case <-time.After(stopDelay):
		case <-ctx.Done():
			return responseMessageID, ctx.Err()
		}
	}

	if err := c.StopStream(ctx, a, sessionID, responseMessageID); err != nil {
		config.Logger.Warn("[fire_completion_and_stop] stop_stream failed", "session_id", sessionID, "message_id", responseMessageID, "error", err)
		return 0, err
	}
	config.Logger.Info("[fire_completion_and_stop] segment sent and stopped", "session_id", sessionID, "response_message_id", responseMessageID, "stop_delay", stopDelay, "account", a.AccountID)
	return responseMessageID, nil
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
