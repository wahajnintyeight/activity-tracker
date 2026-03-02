package cricket

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
)

type MatchState struct {
	TotalRuns         int     `json:"total_runs"`
	Wickets           int     `json:"wickets"`
	Overs             float64 `json:"overs"` // Team's total overs (e.g., 35.5)
	LastScore         string  `json:"last_score"`
	BallsCount        int     `json:"balls_count"`
	BowlerName        string  `json:"bowler_name"`         // Current bowler (e.g., "J. Archer")
	BowlerOvers       float64 `json:"bowler_overs"`        // Bowler's overs (e.g., 2.5)
	BowlerWickets     int     `json:"bowler_wickets"`      // Bowler's wickets taken
	BowlerRunsGiven   int     `json:"bowler_runs_given"`   // Runs conceded by bowler
	BowlerEconomy     string  `json:"bowler_economy"`      // Economy rate (e.g., "1:25")
	DeliverySpeed     string  `json:"delivery_speed"`      // Ball speed (e.g., "129.4km/h")
	BatsmanName       string  `json:"batsman_name"`        // Current batsman (e.g., "A. Wasaie")
	BatsmanRuns       int     `json:"batsman_runs"`        // Batsman's runs
	BatsmanBalls      int     `json:"batsman_balls"`       // Balls faced by batsman
	BatsmanStrikeRate float64 `json:"batsman_strike_rate"` // Batsman's strike rate
	DismissalBowler   string  `json:"dismissal_bowler"`    // Bowler who took the wicket
	DismissalFielder  string  `json:"dismissal_fielder"`   // Fielder involved in dismissal (optional)
	DismissalType     string  `json:"dismissal_type"`      // Type: caught, bowled, lbw, run out, etc.
	CareerRuns        int     `json:"career_runs"`         // Batsman's career runs
	CareerAverage     float64 `json:"career_average"`      // Batsman's career average
	CareerMatches     int     `json:"career_matches"`      // Batsman's career matches
	MilestoneType     string  `json:"milestone_type"`      // Milestone achieved (fifty, century, etc.)
	MilestoneRuns     int     `json:"milestone_runs"`      // Runs at milestone
}

type GameEvent struct {
	Type      CricketEventType `json:"type"`
	Payload   string           `json:"payload"`
	Raw       string           `json:"raw"`
	MatchData *MatchState      `json:"match_data,omitempty"`
}

// CricketImagePayload is sent to the queue for LLM-based OCR analysis
type CricketImagePayload struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Image     *image.RGBA `json:"-"`                    // Image data (not serialized directly)
	ImageData []byte      `json:"image_data,omitempty"` // Base64 encoded PNG
}

// ProcessScore analyzes OCR text and detects cricket events
func ProcessScore(currentText string, previous *MatchState) (*GameEvent, *MatchState) {
	currentText = strings.TrimSpace(currentText)

	if currentText == "" {
		return nil, previous
	}

	// Check if this is a milestone screen (batsman reaching 50, 100, etc.)
	if isMilestoneScreen(currentText) {
		milestoneState := parseMilestoneDetails(currentText)

		// Build milestone message
		payload := fmt.Sprintf("**%s** reaches %s! 🎉", milestoneState.BatsmanName, milestoneState.MilestoneType)
		if milestoneState.BatsmanRuns > 0 && milestoneState.BatsmanBalls > 0 {
			payload += fmt.Sprintf("\n%d* runs off %d balls (SR: %.1f)",
				milestoneState.BatsmanRuns,
				milestoneState.BatsmanBalls,
				milestoneState.BatsmanStrikeRate)
		}

		event := &GameEvent{
			Type:      EventTypeMilestone,
			Payload:   payload,
			Raw:       currentText,
			MatchData: milestoneState,
		}

		// Keep previous state but update with milestone info
		if previous != nil {
			newState := *previous
			newState.BatsmanName = milestoneState.BatsmanName
			newState.BatsmanRuns = milestoneState.BatsmanRuns
			newState.BatsmanBalls = milestoneState.BatsmanBalls
			newState.BatsmanStrikeRate = milestoneState.BatsmanStrikeRate
			newState.MilestoneType = milestoneState.MilestoneType
			newState.MilestoneRuns = milestoneState.MilestoneRuns
			newState.LastScore = currentText
			return event, &newState
		}
		return event, milestoneState
	}

	// Check if this is a wicket dismissal screen (batsman departing)
	if isWicketDismissalScreen(currentText) {
		dismissalState := parseDismissalDetails(currentText)

		// Build detailed dismissal message
		payload := fmt.Sprintf("Batsman dismissed: %s scored %d runs off %d balls (SR: %.1f)",
			dismissalState.BatsmanName,
			dismissalState.BatsmanRuns,
			dismissalState.BatsmanBalls,
			dismissalState.BatsmanStrikeRate)

		if dismissalState.DismissalType != "" {
			payload += fmt.Sprintf(" | Dismissal: %s", dismissalState.DismissalType)
			if dismissalState.DismissalFielder != "" {
				payload += fmt.Sprintf(" (c. %s)", dismissalState.DismissalFielder)
			}
			if dismissalState.DismissalBowler != "" {
				payload += fmt.Sprintf(" b. %s", dismissalState.DismissalBowler)
			}
		}

		if dismissalState.Wickets > 0 && dismissalState.TotalRuns > 0 {
			payload += fmt.Sprintf(" | Score: %d/%d", dismissalState.Wickets, dismissalState.TotalRuns)
		}

		event := &GameEvent{
			Type:      EventTypeBatsmanDepart,
			Payload:   payload,
			Raw:       currentText,
			MatchData: dismissalState,
		}

		// Keep previous state but merge with dismissal details
		if previous != nil {
			newState := *previous
			newState.BatsmanName = dismissalState.BatsmanName
			newState.BatsmanRuns = dismissalState.BatsmanRuns
			newState.BatsmanBalls = dismissalState.BatsmanBalls
			newState.BatsmanStrikeRate = dismissalState.BatsmanStrikeRate
			newState.DismissalBowler = dismissalState.DismissalBowler
			newState.DismissalFielder = dismissalState.DismissalFielder
			newState.DismissalType = dismissalState.DismissalType
			newState.Wickets = dismissalState.Wickets
			newState.TotalRuns = dismissalState.TotalRuns
			newState.LastScore = currentText
			return event, &newState
		}
		return event, dismissalState
	}

	// Check if this is a bowler stats screen (new bowler arriving)
	if isBowlerStatsScreen(currentText) {
		bowlerStats := parseBowlerCareerStats(currentText)

		// Build arrival message with career stats
		payload := fmt.Sprintf("New bowler: %s", bowlerStats.BowlerName)
		if bowlerStats.CareerMatches > 0 {
			payload += fmt.Sprintf(" | Career: %d matches, %d wickets",
				bowlerStats.CareerMatches,
				bowlerStats.BowlerWickets)
		}

		event := &GameEvent{
			Type:      EventTypeBowlerArrive,
			Payload:   payload,
			Raw:       currentText,
			MatchData: bowlerStats,
		}

		// Keep previous state but update bowler info
		if previous != nil {
			newState := *previous
			newState.BowlerName = bowlerStats.BowlerName
			newState.BowlerWickets = bowlerStats.BowlerWickets
			newState.CareerMatches = bowlerStats.CareerMatches
			newState.LastScore = currentText
			return event, &newState
		}
		return event, bowlerStats
	}

	// Check if this is a batsman stats screen (new batsman arriving)
	if isBatsmanStatsScreen(currentText) {
		careerStats := parseBatsmanCareerStats(currentText)

		// Build arrival message with career stats
		payload := fmt.Sprintf("New batsman: %s", careerStats.BatsmanName)
		if careerStats.CareerMatches > 0 {
			payload += fmt.Sprintf(" | Career: %d matches, %d runs, avg %.2f",
				careerStats.CareerMatches,
				careerStats.CareerRuns,
				careerStats.CareerAverage)
		}

		event := &GameEvent{
			Type:      EventTypeBatsmanArrive,
			Payload:   payload,
			Raw:       currentText,
			MatchData: careerStats,
		}

		// Keep previous state but update batsman info
		if previous != nil {
			newState := *previous
			newState.BatsmanName = careerStats.BatsmanName
			newState.CareerRuns = careerStats.CareerRuns
			newState.CareerAverage = careerStats.CareerAverage
			newState.CareerMatches = careerStats.CareerMatches
			newState.LastScore = currentText
			return event, &newState
		}
		return event, careerStats
	}

	// Initialize previous state if nil
	if previous == nil {
		return nil, &MatchState{LastScore: currentText}
	}

	// If no change, return nil event
	if currentText == previous.LastScore {
		return nil, previous
	}

	// Parse current score
	currentState := parseScoreText(currentText)
	if currentState == nil {
		// Couldn't parse, keep previous state
		return nil, previous
	}

	// Detect event based on state change
	event := detectEvent(previous, currentState)

	// Update last score
	currentState.LastScore = currentText

	return event, currentState
}

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
// Format: "MUHAMMAD WAHAJ MINUTES 24 101* BALLS 24 STRIKE RATE 420.8 4 6s 14 MEM 62 7033"
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

// parseMilestoneDetails extracts milestone information from the screen
// Format: "MUHAMMAD WAHAJ MINUTES 24 101* BALLS 24 STRIKE RATE 420.8 4 6s 14 MEM 62 7033"
func parseMilestoneDetails(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	// Extract batsman name (before MINUTES)
	minutesIdx := strings.Index(textLower, "minutes")
	if minutesIdx > 0 {
		beforeMinutes := strings.TrimSpace(text[:minutesIdx])
		lines := strings.Split(beforeMinutes, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if len(line) > 2 && len(line) < 50 {
				hasLetters := false
				for _, ch := range line {
					if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
						hasLetters = true
						break
					}
				}
				if hasLetters {
					state.BatsmanName = correctPlayerName(strings.Title(strings.ToLower(line)))
					break
				}
			}
		}
	}

	// Extract runs with '*' (e.g., "101*")
	runsPattern := regexp.MustCompile(`(\d+)\*`)
	if matches := runsPattern.FindStringSubmatch(text); len(matches) >= 2 {
		runs, _ := strconv.Atoi(matches[1])
		state.BatsmanRuns = runs
		state.MilestoneRuns = runs

		// Determine milestone type based on runs
		state.MilestoneType = determineMilestoneType(runs)
	}

	// Extract balls faced: "BALLS 24"
	ballsPattern := regexp.MustCompile(`balls\s+(\d+)`)
	if matches := ballsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanBalls, _ = strconv.Atoi(matches[1])
	}

	// Extract strike rate: "STRIKE RATE 420.8"
	strikeRatePattern := regexp.MustCompile(`strike rate\s+(\d+\.?\d*)`)
	if matches := strikeRatePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanStrikeRate, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

// determineMilestoneType determines the milestone type based on runs scored
func determineMilestoneType(runs int) string {
	switch {
	case runs >= 50 && runs <= 58:
		return "Half-Century"
	case runs >= 100 && runs <= 108:
		return "Century"
	case runs >= 150 && runs <= 158:
		return "150"
	case runs >= 200 && runs <= 208:
		return "Double Century"
	case runs >= 250 && runs <= 258:
		return "250"
	case runs >= 300 && runs <= 308:
		return "Triple Century"
	default:
		return fmt.Sprintf("%d runs", runs)
	}
}

// extractDismissedBatsmanName extracts the batsman name from wicket dismissal screen
// The name appears before "MINUTES" or "RUN OUT" keywords
func extractDismissedBatsmanName(text string) string {
	textLower := strings.ToLower(text)

	// Try to find "MINUTES" first
	minutesIdx := strings.Index(textLower, "minutes")
	if minutesIdx == -1 {
		// Try "RUN OUT"
		runOutIdx := strings.Index(textLower, "run out")
		if runOutIdx != -1 {
			minutesIdx = runOutIdx
		}
	}

	if minutesIdx == -1 {
		return "Unknown Batsman"
	}

	// Extract everything before the keyword
	beforeKeyword := strings.TrimSpace(text[:minutesIdx])

	// Split by newlines and get the last line (closest to keyword)
	lines := strings.Split(beforeKeyword, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Skip lines with dismissal details (c., b., lbw)
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, " c. ") ||
			strings.Contains(lineLower, " b. ") ||
			strings.Contains(lineLower, "lbw") ||
			strings.Contains(lineLower, "fall of wicket") {
			continue
		}

		if len(line) > 2 && len(line) < 50 {
			// Check if it contains letters
			hasLetters := false
			for _, ch := range line {
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					hasLetters = true
					break
				}
			}
			if hasLetters {
				return correctPlayerName(strings.Title(strings.ToLower(line)))
			}
		}
	}

	return "Unknown Batsman"
}

// parseDismissalDetails extracts detailed dismissal information from wicket screen
// Format: "MARNI-JS LABUSCHAGNE MINUTES C. D WARNER b. S. AFRIDI 6s BALLS 11 STRIKE RATE 118.2 13 FALL OF WICKET 7/51"
func parseDismissalDetails(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	// Extract batsman name (before MINUTES)
	state.BatsmanName = extractDismissedBatsmanName(text)

	// Extract dismissal type and players
	// Pattern: "c. FIELDER b. BOWLER" or "b. BOWLER" or "lbw b. BOWLER" or "RUN OUT (FIELDER)"

	// Check for caught dismissal: "c. FIELDER b. BOWLER"
	caughtPattern := regexp.MustCompile(`c\.?\s+([a-z\.\s]+?)\s+b\.?\s+([a-z\.\s]+?)(?:\s+\d+s|\s+balls|$)`)
	if matches := caughtPattern.FindStringSubmatch(textLower); len(matches) >= 3 {
		state.DismissalType = "caught"
		state.DismissalFielder = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		state.DismissalBowler = correctPlayerName(strings.Title(strings.TrimSpace(matches[2])))
	} else {
		// Check for bowled: "b. BOWLER" (before 6s or balls)
		bowledPattern := regexp.MustCompile(`b\.?\s+([a-z\.\s]+?)(?:\s+\d+s|\s+balls|$)`)
		if matches := bowledPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
			state.DismissalType = "bowled"
			state.DismissalBowler = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}
	}

	// Check for run out: "RUN OUT (FIELDER)"
	if strings.Contains(textLower, "run out") {
		state.DismissalType = "run out"
		runOutPattern := regexp.MustCompile(`run out\s*\(([a-z\.\s]+)\)`)
		if matches := runOutPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
			state.DismissalFielder = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}
	}

	// Check for LBW
	if strings.Contains(textLower, "lbw") {
		state.DismissalType = "lbw"
	}

	// Extract balls faced: "BALLS 11"
	ballsPattern := regexp.MustCompile(`balls\s+(\d+)`)
	if matches := ballsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanBalls, _ = strconv.Atoi(matches[1])
	}

	// Extract strike rate: "STRIKE RATE 118.2"
	strikeRatePattern := regexp.MustCompile(`strike rate\s+(\d+\.?\d*)`)
	if matches := strikeRatePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanStrikeRate, _ = strconv.ParseFloat(matches[1], 64)
	}

	// Extract batsman runs: number before "FALL OF WICKET"
	// Pattern: "118.2 13 FALL OF WICKET" - the 13 is the runs
	fallOfWicketIdx := strings.Index(textLower, "fall of wicket")
	if fallOfWicketIdx > 0 {
		beforeFall := textLower[:fallOfWicketIdx]
		// Find the last number before "fall of wicket"
		runsPattern := regexp.MustCompile(`(\d+)\s*$`)
		if matches := runsPattern.FindStringSubmatch(beforeFall); len(matches) >= 2 {
			state.BatsmanRuns, _ = strconv.Atoi(matches[1])
		}
	}

	// Extract team score at fall of wicket: "FALL OF WICKET 7/51"
	fowPattern := regexp.MustCompile(`fall of wicket\s+(\d+)[/-](\d+)`)
	if matches := fowPattern.FindStringSubmatch(textLower); len(matches) >= 3 {
		state.Wickets, _ = strconv.Atoi(matches[1])
		state.TotalRuns, _ = strconv.Atoi(matches[2])
	}

	return state
}

// extractBowlerNameFromStats extracts the bowler name from stats screen
// The name appears before "MATCHES" keyword
func extractBowlerNameFromStats(text string) string {
	textLower := strings.ToLower(text)

	// Find the position of "MATCHES" keyword
	matchesIdx := strings.Index(textLower, "matches")
	if matchesIdx == -1 {
		return "Unknown Bowler"
	}

	// Extract everything before "MATCHES"
	beforeMatches := strings.TrimSpace(text[:matchesIdx])

	// Split by newlines and get the last line (closest to MATCHES)
	lines := strings.Split(beforeMatches, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len(line) > 2 && len(line) < 50 {
			// Check if it contains letters
			hasLetters := false
			for _, ch := range line {
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					hasLetters = true
					break
				}
			}
			if hasLetters {
				return correctPlayerName(strings.Title(strings.ToLower(line)))
			}
		}
	}

	return "Unknown Bowler"
}

// extractBatsmanNameFromStats extracts the batsman name from stats screen
// The name is usually at the top of the stats screen
// extractBatsmanNameFromStats extracts the batsman name from stats screen
// The name appears before "MATCHES" keyword
func extractBatsmanNameFromStats(text string) string {
	textLower := strings.ToLower(text)

	// Find the position of "MATCHES" keyword
	matchesIdx := strings.Index(textLower, "matches")
	if matchesIdx == -1 {
		return "Unknown Batsman"
	}

	// Extract everything before "MATCHES"
	beforeMatches := strings.TrimSpace(text[:matchesIdx])

	// Split by newlines and get the last line (closest to MATCHES)
	lines := strings.Split(beforeMatches, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if len(line) > 2 && len(line) < 50 {
			// Check if it contains letters
			hasLetters := false
			for _, ch := range line {
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					hasLetters = true
					break
				}
			}
			if hasLetters {
				return correctPlayerName(strings.Title(strings.ToLower(line)))
			}
		}
	}

	return "Unknown Batsman"
}

// parseBowlerCareerStats extracts career statistics from bowler arrival screen
// Format: "MUHAMMAD WAHAJ MATCHES 17 WICKETS 51 RUNS 930 AVERAGE 18.24 5WI 5 LEFT ARM FAST 10WM 1 BEST 6/63"
func parseBowlerCareerStats(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	// Extract bowler name (before MATCHES)
	state.BowlerName = extractBowlerNameFromStats(text)

	// Extract career matches: "MATCHES 17"
	matchesPattern := regexp.MustCompile(`matches\s+(\d+)`)
	if matches := matchesPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerMatches, _ = strconv.Atoi(matches[1])
	}

	// Extract career wickets: "WICKETS 51"
	wicketsPattern := regexp.MustCompile(`wickets\s+(\d+)`)
	if matches := wicketsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BowlerWickets, _ = strconv.Atoi(matches[1])
	}

	// Extract career runs given: "RUNS 930"
	runsPattern := regexp.MustCompile(`runs\s+(\d+)`)
	if matches := runsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BowlerRunsGiven, _ = strconv.Atoi(matches[1])
	}

	// Extract bowling average: "AVERAGE 18.24"
	averagePattern := regexp.MustCompile(`average\s+(\d+\.?\d*)`)
	if matches := averagePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerAverage, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

// parseBatsmanCareerStats extracts career statistics from batsman arrival screen
// Format: "JASPRIT BUMRAH MATCHES 195 RIGHT HAND BAT HUNDREDS RUNS 101 AVERAGE 7.77 STRIKE RATE 81.5 FIFTIES HIGH SCORE"
func parseBatsmanCareerStats(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	// Extract batsman name (before MATCHES)
	state.BatsmanName = extractBatsmanNameFromStats(text)

	// Extract career matches: "MATCHES 195"
	matchesPattern := regexp.MustCompile(`matches\s+(\d+)`)
	if matches := matchesPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerMatches, _ = strconv.Atoi(matches[1])
	}

	// Extract career runs: "RUNS 101"
	runsPattern := regexp.MustCompile(`runs\s+(\d+)`)
	if matches := runsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerRuns, _ = strconv.Atoi(matches[1])
	}

	// Extract career average: "AVERAGE 7.77"
	averagePattern := regexp.MustCompile(`average\s+(\d+\.?\d*)`)
	if matches := averagePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerAverage, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

// parseScoreText extracts runs, wickets, and overs from OCR text
// Expected formats from Cricket 24:
// - Multi-line format with player names on separate lines
// - Bottom bar: "EE 8/188 35.5 OVERS FOOTWORK TIMING SHOT CHOICE J. Archer 1:25 (2.5) Delivery speed 129.4km/h"
// - Score pattern: "8/188", "35.5 OVERS", "J. Archer 1:25 (2.5)"
// parseScoreText extracts runs, wickets, and overs from OCR text
// Expected formats from Cricket 24:
// - Multi-line format with player names on separate lines
// - Bottom bar: "OGS 6/180 27.3 OVERS ... J. Archer 9.1 ... A. Wasaie 0 11 ... 87km/h"
// - Score pattern: "6/180" (wickets/runs), "27.3 OVERS", "A. Wasaie 0 11" (name runs balls)
func parseScoreText(text string) *MatchState {
	originalText := text

	state := &MatchState{}

	// Split into lines for multi-line OCR output
	lines := strings.Split(text, "\n")

	// Process each line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Pattern 1: Team Score - Wickets/Runs (e.g., "6/180")
		scorePattern := regexp.MustCompile(`(\d+)[/-](\d+)`)
		if matches := scorePattern.FindStringSubmatch(lineLower); len(matches) >= 3 {
			state.Wickets, _ = strconv.Atoi(matches[1])
			state.TotalRuns, _ = strconv.Atoi(matches[2])
		}

		// Pattern 2: Team overs (e.g., "27.3 overs")
		oversPattern := regexp.MustCompile(`(\d+\.?\d*)\s*overs?`)
		if matches := oversPattern.FindStringSubmatch(lineLower); len(matches) >= 2 {
			state.Overs, _ = strconv.ParseFloat(matches[1], 64)
		}

		// Pattern 3: Bowler overs in brackets (e.g., "(9.1)")
		bowlerOversPattern := regexp.MustCompile(`\((\d+\.?\d*)\)`)
		if matches := bowlerOversPattern.FindStringSubmatch(lineLower); len(matches) >= 2 {
			state.BowlerOvers, _ = strconv.ParseFloat(matches[1], 64)
		}

		// Pattern 4a: Bowler name with wickets-runs format (e.g., "s. thakur 0-30")
		// This is the format: name wickets-runs (overs)
		bowlerWicketsRunsPattern := regexp.MustCompile(`([a-z]\.?\s+[a-z]+)\s+(\d+)-(\d+)`)
		if matches := bowlerWicketsRunsPattern.FindStringSubmatch(lineLower); len(matches) >= 4 {
			state.BowlerName = correctPlayerName(strings.Title(matches[1]))
			state.BowlerWickets, _ = strconv.Atoi(matches[2])
			state.BowlerRunsGiven, _ = strconv.Atoi(matches[3])
		}

		// Pattern 4b: Bowler name and economy (e.g., "j. archer 1:25")
		// Look for pattern: initial.name followed by time-like format
		bowlerPattern := regexp.MustCompile(`([a-z]\.?\s+[a-z]+)\s+(\d+:\d+)`)
		if matches := bowlerPattern.FindStringSubmatch(lineLower); len(matches) >= 3 {
			// Only set if not already set by wickets-runs pattern
			if state.BowlerName == "" {
				state.BowlerName = correctPlayerName(strings.Title(matches[1]))
			}
			state.BowlerEconomy = matches[2]
		}

		// Pattern 5: Batsman stats - Name followed by runs and balls (e.g., "a. wasaie 0 11")
		// This captures: [initial.]name runs balls
		batsmanStatsPattern := regexp.MustCompile(`([a-z]\.?[a-z\s]+)\s+(\d+)\s+(\d+)`)
		if matches := batsmanStatsPattern.FindStringSubmatch(lineLower); len(matches) >= 4 {
			state.BatsmanName = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
			state.BatsmanRuns, _ = strconv.Atoi(matches[2])
			state.BatsmanBalls, _ = strconv.Atoi(matches[3])
		}

		// Pattern 5b: Batsman with > separator (e.g., "m. labuschagne > c. green 0 0")
		batsmanSeparatorPattern := regexp.MustCompile(`([a-z]\.?[a-z\s]+)\s*>\s*([a-z]\.?[a-z\s]+)`)
		if matches := batsmanSeparatorPattern.FindStringSubmatch(lineLower); len(matches) >= 3 {
			// First name is usually the striker
			state.BatsmanName = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}

		// Pattern 6: Player name with initial (e.g., "j. archer", "a. wasaie")
		// Look for lines that contain a period followed by a space and name
		playerNamePattern := regexp.MustCompile(`^([a-z]\.?\s+[a-z]+)$`)
		if matches := playerNamePattern.FindStringSubmatch(lineLower); len(matches) >= 2 {
			// This might be a standalone player name line
			name := correctPlayerName(strings.Title(matches[1]))
			if state.BowlerName == "" && state.BatsmanName == "" {
				// Could be either bowler or batsman, store as batsman for now
				state.BatsmanName = name
			}
		}

		// Pattern 8: Wicket Summary (Specific to Cricket 24 "FALL OF WICKET" screen)
		// If "FALL OF WICKET" or "MINUTES BALLS" is found, try to find the player name at the beginning of the text
		if strings.Contains(lineLower, "fall of wicket") || strings.Contains(lineLower, "minutes balls") {
			// Look at previous lines or the start of this line for a potential name
			// In Cricket 24 summary, the name is often at the very top
			potentialNamePattern := regexp.MustCompile(`^([a-z\s]{5,25})`)
			if nameMatches := potentialNamePattern.FindStringSubmatch(lineLower); len(nameMatches) >= 2 {
				name := strings.TrimSpace(nameMatches[1])
				// Filter out HUD labels
				if !strings.Contains(strings.ToLower(name), "minutes") && !strings.Contains(strings.ToLower(name), "balls") {
					state.BatsmanName = correctPlayerName(strings.Title(name))
				}
			}
		}

		// Pattern 7: Delivery speed (e.g., "129.4km/h" or "87km/h")
		speedPattern := regexp.MustCompile(`(\d+\.?\d*)\s*km/h`)
		if matches := speedPattern.FindStringSubmatch(lineLower); len(matches) >= 2 {
			state.DeliverySpeed = matches[1] + " km/h"
		}
	}

	state.LastScore = originalText

	// Return nil if we couldn't parse basic score
	if state.TotalRuns == 0 && state.Wickets == 0 {
		return nil
	}

	return state
}

// correctPlayerName fixes common OCR misreadings of player names
func correctPlayerName(name string) string {
	// Common OCR corrections
	corrections := map[string]string{
		"M. Wanaj":             "M. Wahaj",
		"M. wanaj":             "M. Wahaj",
		"wanaj":                "Wahaj",
		"Wanaj":                "Wahaj",
		"Muhammad Wahaj":       "Muhammad Wahaj",
		"muhammad wahaj":       "Muhammad Wahaj",
		"S. Inakur":            "S. Thakur",
		"Inakur":               "Thakur",
		"Jasprit Bumrah":       "Jasprit Bumrah",
		"jasprit bumrah":       "Jasprit Bumrah",
		"Cameron Green":        "Cameron Green",
		"cameron green":        "Cameron Green",
		"Kuldeep Yadav":        "Kuldeep Yadav",
		"kuldeep yadav":        "Kuldeep Yadav",
		"M. Rizwan":            "M. Rizwan",
		"m. rizwan":            "M. Rizwan",
		"M. Starc":             "M. Starc",
		"m. starc":             "M. Starc",
		"Marni-Js Labuschagne": "Marnus Labuschagne",
		"marni-js labuschagne": "Marnus Labuschagne",
		"Marnus Labuschagne":   "Marnus Labuschagne",
		"marnus labuschagne":   "Marnus Labuschagne",
		"D Warner":             "D. Warner",
		"d warner":             "D. Warner",
		"D. Warner":            "D. Warner",
		"S. Afridi":            "S. Afridi",
		"s. afridi":            "S. Afridi",
	}

	// Check for exact match
	if corrected, ok := corrections[name]; ok {
		return corrected
	}

	// Check for partial match (case-insensitive)
	nameLower := strings.ToLower(name)
	for wrong, right := range corrections {
		if strings.Contains(nameLower, strings.ToLower(wrong)) {
			return strings.Replace(name, wrong, right, 1)
		}
	}

	return name
}

// detectEvent compares previous and current state to identify cricket events
// detectEvent compares previous and current state to identify cricket events
func detectEvent(prev, curr *MatchState) *GameEvent {
	if curr == nil {
		return nil
	}

	// If no previous state, don't detect any events (just initialize)
	// This prevents false positives when first detecting the scoreboard
	if prev == nil {
		return nil
	}

	// If previous state had no score data, don't detect events yet
	if prev.TotalRuns == 0 && prev.Wickets == 0 {
		return nil
	}

	runsDiff := curr.TotalRuns - prev.TotalRuns
	wicketsDiff := curr.Wickets - prev.Wickets

	// Wicket fallen - only if wickets increased AND runs also changed
	// This prevents false positives from OCR variations
	if wicketsDiff > 0 && (runsDiff >= 0) {
		return &GameEvent{
			Type:      EventTypeWicket,
			Payload:   fmt.Sprintf("Wicket! %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs),
			Raw:       curr.LastScore,
			MatchData: curr,
		}
	}

	// Boundary detection
	if runsDiff == 6 {
		payload := fmt.Sprintf("SIX! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
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

	if runsDiff == 4 {
		payload := fmt.Sprintf("FOUR! Score: %d/%d (Overs: %.1f)", curr.Wickets, curr.TotalRuns, curr.Overs)
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

	// Regular runs (1, 2, 3, or 5)
	if runsDiff > 0 && runsDiff < 4 {
		return &GameEvent{
			Type:      EventTypeRuns,
			Payload:   fmt.Sprintf("%d run(s). Score: %d/%d (Overs: %.1f)", runsDiff, curr.Wickets, curr.TotalRuns, curr.Overs),
			Raw:       curr.LastScore,
			MatchData: curr,
		}
	}

	// No significant event
	return nil
}
