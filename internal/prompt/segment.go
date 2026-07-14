package prompt

import (
	"regexp"
	"strings"
)

var roleMarkerPattern = regexp.MustCompile(`<System>:|<User>:|<Assistant>:|<Tool>:`)

type promptBlock struct {
	marker string
	text   string
}

func splitIntoBlocks(prompt string) []promptBlock {
	if prompt == "" {
		return nil
	}
	indices := roleMarkerPattern.FindAllStringIndex(prompt, -1)
	if len(indices) == 0 {
		return []promptBlock{{marker: "", text: prompt}}
	}

	blocks := make([]promptBlock, 0, len(indices)+1)

	if indices[0][0] > 0 {
		blocks = append(blocks, promptBlock{marker: "", text: prompt[:indices[0][0]]})
	}

	for i, loc := range indices {
		start := loc[0]
		var end int
		if i+1 < len(indices) {
			end = indices[i+1][0]
		} else {
			end = len(prompt)
		}
		segment := prompt[start:end]
		marker := segment[:loc[1]-loc[0]]
		text := segment[loc[1]-loc[0]:]
		blocks = append(blocks, promptBlock{marker: marker, text: text})
	}
	return blocks
}

func blockRunes(b promptBlock) int {
	return len([]rune(b.marker)) + len([]rune(b.text))
}

func blockString(b promptBlock) string {
	return b.marker + b.text
}

// SplitByRoleBoundary splits a finalized prompt into segments where each
// segment's rune count does not exceed maxChars. Splitting happens at role
// marker boundaries (<System>:, <User>:, <Assistant>:, <Tool>:). When a
// single block itself exceeds maxChars it is hard-cut at maxChars rune
// boundaries. Returns a single-element slice when no split is needed.
func SplitByRoleBoundary(prompt string, maxChars int) []string {
	if maxChars <= 0 || len([]rune(prompt)) <= maxChars {
		return []string{prompt}
	}

	blocks := splitIntoBlocks(prompt)
	if len(blocks) == 0 {
		return []string{prompt}
	}

	var segments []string
	var current strings.Builder
	currentRunes := 0

	flushCurrent := func() {
		if currentRunes > 0 {
			segments = append(segments, current.String())
			current.Reset()
			currentRunes = 0
		}
	}

	for _, block := range blocks {
		blockLen := blockRunes(block)

		if currentRunes > 0 && currentRunes+blockLen > maxChars {
			flushCurrent()
		}

		if blockLen <= maxChars {
			current.WriteString(blockString(block))
			currentRunes += blockLen
		} else {
			if currentRunes > 0 {
				flushCurrent()
			}
			full := blockString(block)
			runes := []rune(full)
			for len(runes) > 0 {
				take := maxChars
				if take > len(runes) {
					take = len(runes)
				}
				segments = append(segments, string(runes[:take]))
				runes = runes[take:]
			}
		}
	}
	flushCurrent()

	if len(segments) == 0 {
		return []string{prompt}
	}
	return segments
}
