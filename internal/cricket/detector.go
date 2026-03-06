package cricket

import (
	"fmt"
	"regexp"
	"strconv"
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

// detectEvents compares previous and current state to identify cricket events.
// Multiple events can occur in the same frame (for example: boundary + team milestone).

func parseMatchWonResult(text string) (winner string, margin int, unit string, ok bool) {
	normalized := strings.Join(strings.Fields(text), " ")
	if normalized == "" {
		return "", 0, "", false
	}

	// Examples:
	// TEAMNAME WON BY 200 RUNS
	// TEAMNAME WON BY 8 WICKETS
	re := regexp.MustCompile(`(?i)\b([a-z0-9 .&'\-]+?)\s+won\s+by\s+(\d+)\s+(runs?|wickets?)\b`)
	m := re.FindStringSubmatch(normalized)
	if len(m) != 4 {
		return "", 0, "", false
	}

	winner = strings.TrimSpace(m[1])
	winner = correctPlayerName(strings.Title(strings.ToLower(winner)))

	parsedMargin, err := strconv.Atoi(strings.TrimSpace(m[2]))
	if err != nil || parsedMargin <= 0 {
		return "", 0, "", false
	}
	margin = parsedMargin

	u := strings.ToUpper(strings.TrimSpace(m[3]))
	if strings.HasPrefix(u, "RUN") {
		unit = "RUNS"
	} else {
		unit = "WICKETS"
	}

	return winner, margin, unit, true
}
func detectEvents(prev, curr *MatchState) []*GameEvent {
	if curr == nil || prev == nil {
		return nil
	}

	if curr.ScoreboardFrames < 2 {
		return nil
	}

	// If previous state had no score data, don't detect events yet
	if prev.TotalRuns == 0 && prev.Wickets == 0 {
		return nil
	}

	runsDiff := curr.TotalRuns - prev.TotalRuns
	wicketsDiff := curr.Wickets - prev.Wickets
	events := make([]*GameEvent, 0, 2)

	// Wicket fallen
	if wicketsDiff == 1 && (runsDiff >= 0) {
		payload := fmt.Sprintf("Wicket! %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("Wicket! %s out. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
		}
		events = append(events, &GameEvent{
			Type:      EventTypeWicket,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		})
		return events
	}

	// Boundary Six
	if runsDiff == 6 {
		payload := fmt.Sprintf("SIX! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("SIX! %s smashed it over the rope. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
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
		events = append(events, &GameEvent{
			Type:      EventTypeBoundarySix,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		})
	}

	// Boundary Four
	if runsDiff == 4 {
		payload := fmt.Sprintf("FOUR! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
		if curr.BatsmanName != "" {
			payload = fmt.Sprintf("FOUR! %s has hit the ropes. Score: %d/%d (Overs: %.1f)", curr.BatsmanName, curr.Wickets, curr.TotalRuns, curr.Overs)
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
		events = append(events, &GameEvent{
			Type:      EventTypeBoundaryFour,
			Payload:   payload,
			Raw:       curr.LastScore,
			MatchData: curr,
		})
	}

	// Chase update event from "Need A from B balls"
	if prev.NeedRuns > 0 && curr.NeedRuns > 0 && prev.NeedBalls > 0 && curr.NeedBalls > 0 {
		needDiff := curr.NeedRuns - prev.NeedRuns
		runsReduced := -needDiff
		if needDiff < 0 && (runsReduced == 3 || runsReduced == 4 || runsReduced == 6) {
			status := getChaseStatus(curr.NeedRuns, curr.NeedBalls)
			payload := fmt.Sprintf("Chase update: Need %d from %d balls (%s)", curr.NeedRuns, curr.NeedBalls, status)
			events = append(events, &GameEvent{
				Type:      EventTypeChaseUpdate,
				Payload:   payload,
				Raw:       curr.LastScore,
				MatchData: curr,
			})
		}
	}

	// Team score milestone (50, 100, 150, ...)
	if runsDiff > 0 {
		if milestoneRuns, ok := getCrossedTeamMilestone(prev.TotalRuns, curr.TotalRuns); ok {
			curr.TeamMilestoneRuns = milestoneRuns
			payload := fmt.Sprintf("Team milestone! %d up. Score: %d/%d (Overs: %.1f)", milestoneRuns, curr.Wickets, curr.TotalRuns, curr.Overs)
			events = append(events, &GameEvent{
				Type:      EventTypeTeamMilestone,
				Payload:   payload,
				Raw:       curr.LastScore,
				MatchData: curr,
			})
		}
	}

	if len(events) == 0 {
		return nil
	}
	return events
}

func getCrossedTeamMilestone(prevRuns, currRuns int) (int, bool) {
	if currRuns <= prevRuns {
		return 0, false
	}

	// Explicit threshold crossing rule:
	// trigger when prev < M and curr >= M for M in 50,100,150...
	for milestone := 50; milestone <= currRuns; milestone += 50 {
		if prevRuns < milestone && currRuns >= milestone {
			return milestone, true
		}
	}

	return 0, false
}

func getChaseStatus(needRuns, needBalls int) string {
	if needBalls <= 0 {
		return "no balls left"
	}
	if needRuns <= needBalls {
		return "chase under control"
	}
	gapPct := (float64(needRuns-needBalls) / float64(needBalls)) * 100
	if gapPct <= 10 {
		return "tight chase (within 10%)"
	}
	return "chasing hard"
}
