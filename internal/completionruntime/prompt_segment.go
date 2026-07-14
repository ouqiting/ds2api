package completionruntime

import (
	"time"

	"ds2api/internal/config"
	"ds2api/internal/prompt"
	"ds2api/internal/promptcompat"
)

// ExpertPromptSegmentConfigReader is the minimal interface needed to decide
// whether expert prompt segmentation should run.
type ExpertPromptSegmentConfigReader interface {
	ExpertPromptSegmentEnabled() bool
	ExpertPromptSegmentMaxChars() int
	ExpertPromptSegmentStopDelayMs() int
}

// shouldSegmentExpertPrompt returns segmented prompts when the resolved model
// is expert, segmentation is enabled, and the finalized prompt exceeds the
// configured rune threshold. Returns nil when segmentation is not applicable.
func shouldSegmentExpertPrompt(stdReq promptcompat.StandardRequest, opts Options) []string {
	segmentReader := opts.ExpertPromptSegment
	if segmentReader == nil {
		return nil
	}
	if !segmentReader.ExpertPromptSegmentEnabled() {
		return nil
	}
	modelType, ok := config.GetModelType(stdReq.ResolvedModel)
	if !ok || modelType != "expert" {
		return nil
	}
	maxChars := segmentReader.ExpertPromptSegmentMaxChars()
	if maxChars <= 0 {
		return nil
	}
	promptText := stdReq.FinalPrompt
	if len([]rune(promptText)) <= maxChars {
		return nil
	}
	segments := prompt.SplitByRoleBoundary(promptText, maxChars)
	if len(segments) <= 1 {
		return nil
	}
	return segments
}

func segmentStopDelay(opts Options) time.Duration {
	if opts.ExpertPromptSegment == nil {
		return 0
	}
	ms := opts.ExpertPromptSegment.ExpertPromptSegmentStopDelayMs()
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
