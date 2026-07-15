package completionruntime

import (
	"context"
	"net/http"
	"time"

	"ds2api/internal/assistantturn"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
	"ds2api/internal/promptcompat"
)

func StartCompletionWithSegments(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options, segments []string, stopDelay time.Duration) (StartResult, *assistantturn.OutputError) {
	if len(segments) <= 1 {
		return startCompletionOnce(ctx, ds, a, stdReq, opts)
	}

	maxAttempts := opts.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var prepErr *assistantturn.OutputError
	stdReq, prepErr = prepareCurrentInputFile(ctx, ds, a, stdReq, opts)
	if prepErr != nil {
		return StartResult{Request: stdReq}, prepErr
	}

	sessionID, err := ds.CreateSession(ctx, a, maxAttempts)
	if err != nil {
		return StartResult{Request: stdReq}, authOutputError(a)
	}

	parentMessageID := 0
	for i := 0; i < len(segments)-1; i++ {
		segPow, err := ds.GetPow(ctx, a, maxAttempts)
		if err != nil {
			return StartResult{SessionID: sessionID, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusUnauthorized, Message: "Failed to get PoW (invalid token or unknown error).", Code: "error"}
		}
		segPayload := stdReq.CompletionPayloadWithParentAndPrompt(sessionID, parentMessageID, segments[i])
		logSegmentPayload("fire-stop", i, len(segments), sessionID, parentMessageID, segments[i])
		respID, err := ds.FireCompletionAndStop(ctx, a, segPayload, segPow, stopDelay)
		if err != nil {
			if dsclient.IsMutedError(err) {
				return StartResult{SessionID: sessionID, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusForbidden, Message: "Account is muted by upstream.", Code: "account_muted"}
			}
			config.Logger.Warn("[start_completion_with_segments] segment fire-and-stop failed", "segment_index", i, "session_id", sessionID, "parent_message_id", parentMessageID, "error", err)
			return StartResult{SessionID: sessionID, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusInternalServerError, Message: "Failed to send segment before stop: " + err.Error(), Code: "error"}
		}
		parentMessageID = respID
		waitForStoppedSegmentSettle(ctx, sessionID, parentMessageID, stopDelay)
	}

	finalPow, err := ds.GetPow(ctx, a, maxAttempts)
	if err != nil {
		return StartResult{SessionID: sessionID, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusUnauthorized, Message: "Failed to get PoW (invalid token or unknown error).", Code: "error"}
	}
	finalPayload := stdReq.CompletionPayloadWithParentAndPrompt(sessionID, parentMessageID, segments[len(segments)-1])
	logSegmentPayload("final", len(segments)-1, len(segments), sessionID, parentMessageID, segments[len(segments)-1])
	resp, err := ds.CallCompletion(ctx, a, finalPayload, finalPow, maxAttempts)
	if err != nil {
		if dsclient.IsMutedError(err) {
			return StartResult{SessionID: sessionID, Payload: finalPayload, Pow: finalPow, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusForbidden, Message: "Account is muted by upstream.", Code: "account_muted"}
		}
		return StartResult{SessionID: sessionID, Payload: finalPayload, Pow: finalPow, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusInternalServerError, Message: "Failed to get completion.", Code: "error"}
	}
	return StartResult{SessionID: sessionID, Payload: finalPayload, Pow: finalPow, Response: resp, Request: stdReq}, nil
}

func waitForStoppedSegmentSettle(ctx context.Context, sessionID string, parentMessageID int, stopDelay time.Duration) {
	if stopDelay <= 0 {
		return
	}
	settleDelay := 1 * time.Second
	if stopDelay > settleDelay {
		settleDelay = stopDelay
	}
	config.Logger.Info("[start_completion_with_segments] waiting for stopped segment settle", "session_id", sessionID, "parent_message_id", parentMessageID, "settle_delay", settleDelay)
	select {
	case <-time.After(settleDelay):
	case <-ctx.Done():
		config.Logger.Warn("[start_completion_with_segments] stopped segment settle interrupted", "session_id", sessionID, "parent_message_id", parentMessageID, "error", ctx.Err())
	}
}

func logSegmentPayload(kind string, index int, total int, sessionID string, parentMessageID int, prompt string) {
	config.Logger.Info("[start_completion_with_segments] sending segment",
		"kind", kind,
		"segment_index", index,
		"segment_total", total,
		"session_id", sessionID,
		"parent_message_id", parentMessageID,
		"prompt_runes", len([]rune(prompt)),
	)
}
