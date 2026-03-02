package cricket

import (
	"fmt"
	"strings"
)

// isBowlerStatsScreen detects if the OCR text is showing bowler statistics
// Indicators: MATCHES, WICKETS, BEST, 5WI, 10WM (bowling-specific keywords)
func isBowlerStatsScreen(text string) bool {
	textLower := strings.ToLower(text)

	// Must have MATCHES keyword
	if !strings.Contains(textLower, "matches") {
		return false
	}

	// Count bowling-specific keywords
	bowlingKeywords := []string{"wickets", "best", "5wi", "10wm", "left arm", "right arm", "fast", "medium", "spin"}
	matchCount := 0

	for _, keyword := range bowlingKeywords {
		if strings.Contains(textLower, keyword) {
			matchCount++
		}
	}

	// If we have 2 or more bowling keywords along with MATCHES, it's a bowler stats screen
	return matchCount >= 2
}

// isBatsmanStatsScreen detects if the OCR text is showing batsman statistics
// Indicators: MATCHES, HUNDREDS, FIFTIES, HIGH SCORE, RUNS, AVERAGE, STRIKE RATE
func isBatsmanStatsScreen(text string) bool {
	textLower := strings.ToLower(text)

	// Must have MATCHES keyword
	if !strings.Contains(textLower, "matches") {
		return false
	}

	// Count how many stat keywords are present
	keywords := []string{"hundreds", "fifties", "high score", "average", "strike rate"}
	matchCount := 0

	for _, keyword := range keywords {
		if strings.Contains(textLower, keyword) {
			matchCount++
		}
	}

	// If we have 3 or more stat keywords, it's likely a batsman stats screen
	return matchCount >= 3
}

// isWicketDismissalScreen detects if the OCR text is showing wicket dismissal details
// Indicators: MINUTES, BALLS, FALL OF WICKET, RUN OUT, or dismissal types (c., b., lbw)
func isWicketDismissalScreen(text string) bool {
	textLower := strings.ToLower(text)

	// Must have "FALL OF WICKET" to be a dismissal screen
	if strings.Contains(textLower, "fall of wicket") {
		return true
	}

	// Check for run out
	if strings.Contains(textLower, "run out") {
		return true
	}

	// Check for dismissal types with BALLS keyword (common pattern)
	// But only if it also has dismissal indicators
	if strings.Contains(textLower, "balls") &&
		(strings.Contains(textLower, " c. ") ||
			strings.Contains(textLower, " b. ") ||
			strings.Contains(textLower, "lbw")) {
		// Additional check: should have "minutes" for dismissal screen
		if strings.Contains(textLower, "minutes") {
			return true
		}
	}

	return false
}

// isMilestoneScreen detects if the OCR text is showing a batsman milestone
// Indicators: MINUTES, BALLS, STRIKE RATE, but NO "FALL OF WICKET" and has '*' before BALLS
func isMilestoneScreen(text string) bool {
	textLower := strings.ToLower(text)

	// Must have MINUTES and BALLS
	if !strings.Contains(textLower, "minutes") || !strings.Contains(textLower, "balls") {
		return false
	}

	// Must NOT have "FALL OF WICKET"
	if strings.Contains(textLower, "fall of wicket") {
		return false
	}

	// Must have '*' indicating not out milestone
	if !strings.Contains(text, "*") {
		return false
	}

	// Must have STRIKE RATE
	if !strings.Contains(textLower, "strike rate") {
		return false
	}

	return true
}

// detectEvent compares previous and current state to identify cricket events
func detectEvent(prev, curr *MatchState) *GameEvent {
	if curr == nil || prev == nil {
		return nil
	}

	// If previous state had no score data, don't detect events yet
	if prev.TotalRuns == 0 && prev.Wickets == 0 {
		return nil
	}

	runsDiff := curr.TotalRuns - prev.TotalRuns
	wicketsDiff := curr.Wickets - prev.Wickets

	// Wicket fallen
	if wicketsDiff > 0 && (runsDiff >= 0) {
		payload := fmt.Sprintf("Wicket! %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("Wicket! %s out. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
		}
		return &GameEvent{
			Type:      EventTypeWicket,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		}
	}

	// Boundary Six
	if runsDiff == 6 {
		payload := fmt.Sprintf("SIX! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("SIX! %s smash it. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
		}
		if curr.BowlerName != "" {
			bowlerStats := fmt.Sprintf("%.1f overs", curr.BowlerOvers)
			if curr.BowlerWickets > 0 || curr.BowlerRunsGiven > 0 {
				bowlerStats = fmt.Sprintf("%d-%d (%.1f)", curr.BowlerWickets, curr.BowlerRunsGiven, curr.BowlerOvers)
			}
			payload += fmt.Sprintf(" | Bowler: %s %s", curr.BowlerName, bowlerStats)
		}
		if curr.DeliverySpeed != "" {
			payload += fmt.Sprintf(" | Speed: %s", curr.DeliverySpeed)
		}
		return &GameEvent{
			Type:      EventTypeBoundarySix,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		}
	}

	// Boundary Four
	if runsDiff == 4 {
		payload := fmt.Sprintf("FOUR! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("FOUR! %s strike it. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
		}
		if curr.BowlerName != "" {
			bowlerStats := fmt.Sprintf("%.1f overs", curr.BowlerOvers)
			if curr.BowlerWickets > 0 || curr.BowlerRunsGiven > 0 {
				bowlerStats = fmt.Sprintf("%d-%d (%.1f)", curr.BowlerWickets, curr.BowlerRunsGiven, curr.BowlerOvers)
			}
			payload += fmt.Sprintf(" | Bowler: %s %s", curr.BowlerName, bowlerStats)
		}
		if curr.DeliverySpeed != "" {
			payload += fmt.Sprintf(" | Speed: %s", curr.DeliverySpeed)
		}
		return &GameEvent{
			Type:      EventTypeBoundaryFour,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		}
	}

	return nil
}
