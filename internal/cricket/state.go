package cricket

import (
	"fmt"
	"image"
	"regexp"
	"strings"
)

type GameType string

const (
	GameTypeC24 GameType = "c24"
	GameTypeC26 GameType = "c26"
)

type MatchState struct {
	TotalRuns         int     `json:"total_runs"`
	Wickets           int     `json:"wickets"`
	Overs             float64 `json:"overs"` // Team's total overs (e.g., 35.5)
	LastScore         string  `json:"last_score"`
	ScoreboardFrames  int     `json:"scoreboard_frames"`
	BallsCount        int     `json:"balls_count"`
	BowlerName        string  `json:"bowler_name"`         // Current bowler (e.g., "J. Archer")
	BowlerOvers       float64 `json:"bowler_overs"`        // Bowler's overs (e.g., 2.5)
	BowlerWickets     int     `json:"bowler_wickets"`      // Bowler's wickets taken
	BowlerRunsGiven   int     `json:"bowler_runs_given"`   // Runs conceded by bowler
	BowlerEconomy     string  `json:"bowler_economy"`      // Economy rate (e.g., "1:25")
	DeliverySpeed     string  `json:"delivery_speed"`      // Ball speed (e.g., "129.4km/h")
	BatsmanName       string  `json:"batsman_name"`        // Current active striker
	BatsmanLeft       string  `json:"batsman_left"`        // Persisted name of left batsman
	BatsmanRight      string  `json:"batsman_right"`       // Persisted name of right batsman
	BatsmanRuns       int     `json:"batsman_runs"`        // Batsman's runs
	BatsmanBalls      int     `json:"batsman_balls"`       // Balls faced by batsman
	BatsmanStrikeRate float64 `json:"batsman_strike_rate"` // Batsman's strike rate
	IsStrikerOnLeft   bool    `json:"is_striker_on_left"`  // Track striker position using vision
	DismissalBowler   string  `json:"dismissal_bowler"`    // Bowler who took the wicket
	DismissalFielder  string  `json:"dismissal_fielder"`   // Fielder involved in dismissal (optional)
	DismissalType     string  `json:"dismissal_type"`      // Type: caught, bowled, lbw, run out, etc.
	CareerRuns        int     `json:"career_runs"`         // Batsman's career runs
	CareerAverage     float64 `json:"career_average"`      // Batsman's career average
	CareerMatches     int     `json:"career_matches"`      // Batsman's career matches
	MilestoneType     string  `json:"milestone_type"`      // Milestone achieved (fifty, century, etc.)
	MilestoneRuns     int     `json:"milestone_runs"`      // Runs at milestone
	TeamMilestoneRuns int     `json:"team_milestone_runs"` // Team milestone score reached (50, 100, 150, ...)
	NeedRuns          int     `json:"need_runs"`           // Runs required in chase (Need A from B balls)
	NeedBalls         int     `json:"need_balls"`          // Balls remaining in chase
	MatchWinner       string  `json:"match_winner"`        // Team that won the match
	MatchWinMargin    int     `json:"match_win_margin"`    // Winning margin value
	MatchWinType      string  `json:"match_win_type"`      // RUNS or WICKETS
	TeamName          string  `json:"team_name"`           // Current batting team
	OppositionName    string  `json:"opposition_name"`     // Bowling team
	TargetRuns        int     `json:"target_runs"`         // Target runs in 2nd innings
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

// ProcessScoreWithVision analyzes OCR text and detects cricket events with the help of pixel scanning and targeted OCR
func ProcessScoreWithVision(img *image.RGBA, currentText string, previous *MatchState, ocr OCRClient, debug bool, gameType GameType, teamScorePosition string) ([]*GameEvent, *MatchState) {
	currentText = strings.TrimSpace(currentText)

	if currentText == "" {
		return nil, previous
	}

	// Initialize state if first run
	var currentState *MatchState
	if previous == nil {
		currentState = &MatchState{LastScore: currentText}
	} else {
		// Clone previous state to maintain memory of batsmen
		temp := *previous
		currentState = &temp
	}

	// MATCH RESULT screen (TEAMNAME WON BY X RUNS/WICKETS)
	if winner, margin, unit, ok := parseMatchWonResult(currentText); ok {
		currentState.MatchWinner = winner
		currentState.MatchWinMargin = margin
		currentState.MatchWinType = unit

		// Suppress duplicate events if the result is exactly the same as the previous frame
		if previous != nil && previous.MatchWinner == winner &&
			previous.MatchWinMargin == margin && previous.MatchWinType == unit {
			currentState.LastScore = currentText
			return nil, currentState
		}

		payload := fmt.Sprintf("%s WON BY %d %s", strings.ToUpper(strings.TrimSpace(winner)), margin, unit)
		event := &GameEvent{
			Type:      EventTypeMatchWon,
			Payload:   payload,
			Raw:       currentText,
			MatchData: currentState,
		}
		currentState.LastScore = currentText
		return []*GameEvent{event}, currentState
	}
	// 1. Check for SPECIAL SCREENS first (Wickets, Arrivals)

	// BATSMAN DEPARTING (Wicket Screen)
	if isWicketDismissalScreen(currentText) {
		dismissalState := parseDismissalDetails(currentText)

		// Logic: Identify who left and clear their slot (left or right).
		// Match by last name because the HUD abbreviates first names (e.g. "M. Marsh" vs "Mitch Marsh").
		departedName := dismissalState.BatsmanName
		if departedName != "" && lastNamesMatch(departedName, currentState.BatsmanLeft) {
			currentState.BatsmanLeft = ""
		} else if departedName != "" && lastNamesMatch(departedName, currentState.BatsmanRight) {
			currentState.BatsmanRight = ""
		}
		// Clear the active striker since the batsman has departed
		currentState.BatsmanName = ""

		payload := fmt.Sprintf("Wicket! %s dismissed for %d. Score: %d/%d",
			dismissalState.BatsmanName, dismissalState.BatsmanRuns, dismissalState.Wickets, dismissalState.TotalRuns)

		event := &GameEvent{
			Type:      EventTypeBatsmanDepart,
			Payload:   payload,
			Raw:       currentText,
			MatchData: dismissalState,
		}
		currentState.LastScore = currentText
		return []*GameEvent{event}, currentState
	}

	// BATSMAN MILESTONE (Milestone Overlay)
	if isMilestoneScreen(currentText) {
		milestoneState := parseMilestoneDetails(currentText)

		// Prefer detected milestone name; otherwise keep known striker context.
		if milestoneState.BatsmanName == "" {
			milestoneState.BatsmanName = currentState.BatsmanName
		}
		if milestoneState.BatsmanName == "" {
			if currentState.IsStrikerOnLeft {
				milestoneState.BatsmanName = currentState.BatsmanLeft
			} else {
				milestoneState.BatsmanName = currentState.BatsmanRight
			}
		}

		if milestoneState.MilestoneRuns > 0 {
			if milestoneState.BatsmanName != "" {
				currentState.BatsmanName = milestoneState.BatsmanName
			}
			currentState.BatsmanRuns = milestoneState.BatsmanRuns
			currentState.BatsmanBalls = milestoneState.BatsmanBalls
			currentState.BatsmanStrikeRate = milestoneState.BatsmanStrikeRate
			currentState.MilestoneRuns = milestoneState.MilestoneRuns
			currentState.MilestoneType = milestoneState.MilestoneType

			payload := fmt.Sprintf(
				"Milestone! %s reaches %s (%d*) in %d balls at SR %.1f",
				milestoneState.BatsmanName,
				milestoneState.MilestoneType,
				milestoneState.MilestoneRuns,
				milestoneState.BatsmanBalls,
				milestoneState.BatsmanStrikeRate,
			)

			event := &GameEvent{
				Type:      EventTypeMilestone,
				Payload:   payload,
				Raw:       currentText,
				MatchData: milestoneState,
			}
			currentState.LastScore = currentText
			return []*GameEvent{event}, currentState
		}
	}

	// BATSMAN ARRIVING (Career Stats Screen)
	if isBatsmanStatsScreen(currentText) {
		arrivalState := parseBatsmanCareerStats(currentText)

		// Logic: Fill the empty slot left by the departed batsman and make them the active striker.
		// Use last-name matching for the duplicate guard (HUD shows "S. Smith", stats show "Steve Smith").
		switch {
		case currentState.BatsmanLeft == "":
			currentState.BatsmanLeft = arrivalState.BatsmanName
			currentState.IsStrikerOnLeft = true
		case currentState.BatsmanRight == "" && !lastNamesMatch(arrivalState.BatsmanName, currentState.BatsmanLeft):
			currentState.BatsmanRight = arrivalState.BatsmanName
			currentState.IsStrikerOnLeft = false
		case lastNamesMatch(arrivalState.BatsmanName, currentState.BatsmanLeft):
			// Arrived batsman matches left slot — update name and set as striker
			currentState.BatsmanLeft = arrivalState.BatsmanName
			currentState.IsStrikerOnLeft = true
		case lastNamesMatch(arrivalState.BatsmanName, currentState.BatsmanRight):
			// Arrived batsman matches right slot — update name and set as striker
			currentState.BatsmanRight = arrivalState.BatsmanName
			currentState.IsStrikerOnLeft = false
		default:
			// Fallback: depart screen was missed — replace the non-striker slot
			if currentState.IsStrikerOnLeft {
				currentState.BatsmanRight = arrivalState.BatsmanName
				currentState.IsStrikerOnLeft = false
			} else {
				currentState.BatsmanLeft = arrivalState.BatsmanName
				currentState.IsStrikerOnLeft = true
			}
		}
		// The arriving batsman becomes the active striker
		currentState.BatsmanName = arrivalState.BatsmanName

		payload := fmt.Sprintf(
			"New batsman: %s | Career: %d matches, %d runs, avg %.2f",
			arrivalState.BatsmanName,
			arrivalState.CareerMatches,
			arrivalState.CareerRuns,
			arrivalState.CareerAverage,
		)

		event := &GameEvent{
			Type:      EventTypeBatsmanArrive,
			Payload:   payload,
			Raw:       currentText,
			MatchData: arrivalState,
		}
		currentState.LastScore = currentText
		return []*GameEvent{event}, currentState
	}

	// BOWLER ARRIVING (Bowler Stats Screen)
	if isBowlerStatsScreen(currentText) {
		bowlerState := parseBowlerCareerStats(currentText)
		currentState.BowlerName = bowlerState.BowlerName

		event := &GameEvent{
			Type:      EventTypeBowlerArrive,
			Payload:   fmt.Sprintf("New bowler: %s (%d wickets in %d matches)", bowlerState.BowlerName, bowlerState.BowlerWickets, bowlerState.CareerMatches),
			Raw:       currentText,
			MatchData: bowlerState,
		}
		currentState.LastScore = currentText
		return []*GameEvent{event}, currentState
	}

	// 2. STANDARD SCOREBOARD PROCESSING
	if previous != nil && currentText == previous.LastScore {
		updateBatsmenAndStrikerFromZones(img, currentState, previous, "", ocr, debug, gameType, teamScorePosition)
		currentState.LastScore = currentText
		return nil, currentState
	}

	scoreboardState := parseScoreText(currentText, gameType)
	if scoreboardState == nil {
		updateBatsmenAndStrikerFromZones(img, currentState, previous, "", ocr, debug, gameType, teamScorePosition)
		currentState.LastScore = currentText
		return nil, currentState
	}

	if previous == nil {
		currentState.ScoreboardFrames = 1
	} else {
		currentState.ScoreboardFrames = previous.ScoreboardFrames + 1
	}

	// Update score-related fields
	currentState.TotalRuns = scoreboardState.TotalRuns
	currentState.Wickets = scoreboardState.Wickets
	currentState.Overs = scoreboardState.Overs
	currentState.BowlerName = scoreboardState.BowlerName
	currentState.DeliverySpeed = scoreboardState.DeliverySpeed

	// Update team names and target if found in standard score
	if scoreboardState.TeamName != "" {
		currentState.TeamName = scoreboardState.TeamName
	}
	if scoreboardState.OppositionName != "" {
		currentState.OppositionName = scoreboardState.OppositionName
	}
	if scoreboardState.TargetRuns > 0 {
		currentState.TargetRuns = scoreboardState.TargetRuns
	}
	if scoreboardState.NeedRuns > 0 {
		currentState.NeedRuns = scoreboardState.NeedRuns
	}
	if scoreboardState.NeedBalls > 0 {
		currentState.NeedBalls = scoreboardState.NeedBalls
	}

	// Detect new innings start: score has reset to 0/0 at the very beginning of an innings
	// (overs 0.0 or 0.1) while the previous state had a real score. Clear old batsmen
	// so they get freshly re-tracked from the HUD zones of the new innings.
	isNewInnings := previous != nil &&
		(previous.TotalRuns > 0 || previous.Wickets > 0) &&
		currentState.TotalRuns == 0 &&
		currentState.Wickets == 0 &&
		(currentState.Overs == 0.0 || currentState.Overs == 0.1)
	if isNewInnings {
		currentState.BatsmanLeft = ""
		currentState.BatsmanRight = ""
		currentState.BatsmanName = ""
		currentState.IsStrikerOnLeft = false
		currentState.TargetRuns = 0
		currentState.NeedRuns = 0
		currentState.NeedBalls = 0
	}

	updateBatsmenAndStrikerFromZones(img, currentState, previous, scoreboardState.BatsmanName, ocr, debug, gameType, teamScorePosition)

	events := detectEvents(previous, currentState)
	currentState.LastScore = currentText

	return events, currentState
}

func updateBatsmenAndStrikerFromZones(img *image.RGBA, currentState *MatchState, previous *MatchState, scoreboardBatsman string, ocr OCRClient, debug bool, gameType GameType, teamScorePosition string) {
	if img == nil || currentState == nil || ocr == nil {
		return
	}

	// Select zone coordinates
	zones := GetZones(gameType, teamScorePosition)

	bounds := img.Bounds()
	if bounds.Dx() < 1000 || bounds.Dy() < 130 {
		return
	}

	strikerDetectedThisFrame := false
	for _, z := range zones {
		rect := image.Rect(z.Rect[0][0], z.Rect[1][0], z.Rect[0][1], z.Rect[1][1])
		sub := img.SubImage(rect).(*image.RGBA)
		zoneText, _ := ocr.ExtractText(sub)

		cleanName := cleanZoneName(zoneText)
		if cleanName != "" {
			if z.Side == "left" && currentState.BatsmanLeft == "" {
				currentState.BatsmanLeft = cleanName
			} else if z.Side == "right" && currentState.BatsmanRight == "" {
				currentState.BatsmanRight = cleanName
			}
		}

		if DetectStriker(sub, z.Side, debug) || hasStrikerIndicator(zoneText) {
			currentState.IsStrikerOnLeft = (z.Side == "left")
			strikerDetectedThisFrame = true
		}
	}

	if containsDigit(currentState.BatsmanLeft) {
		for _, z := range zones {
			if z.Side != "left" {
				continue
			}
			rect := image.Rect(z.Rect[0][0], z.Rect[1][0], z.Rect[0][1], z.Rect[1][1])
			sub := img.SubImage(rect).(*image.RGBA)
			name, _ := ocr.ExtractText(sub)
			cleanName := cleanZoneName(name)
			if cleanName != "" && !containsDigit(cleanName) {
				currentState.BatsmanLeft = cleanName
				break
			}
		}
	}
	if containsDigit(currentState.BatsmanRight) {
		for _, z := range zones {
			if z.Side != "right" {
				continue
			}
			rect := image.Rect(z.Rect[0][0], z.Rect[1][0], z.Rect[0][1], z.Rect[1][1])
			sub := img.SubImage(rect).(*image.RGBA)
			name, _ := ocr.ExtractText(sub)
			cleanName := cleanZoneName(name)
			if cleanName != "" && !containsDigit(cleanName) {
				currentState.BatsmanRight = cleanName
				break
			}
		}
	}

	if strikerDetectedThisFrame {
		if currentState.IsStrikerOnLeft {
			currentState.BatsmanName = currentState.BatsmanLeft
		} else {
			currentState.BatsmanName = currentState.BatsmanRight
		}
		return
	}

	if matchesPlayer(scoreboardBatsman, currentState.BatsmanLeft) {
		currentState.IsStrikerOnLeft = true
		currentState.BatsmanName = currentState.BatsmanLeft
		return
	}
	if matchesPlayer(scoreboardBatsman, currentState.BatsmanRight) {
		currentState.IsStrikerOnLeft = false
		currentState.BatsmanName = currentState.BatsmanRight
		return
	}
	if previous != nil {
		currentState.IsStrikerOnLeft = previous.IsStrikerOnLeft
		if currentState.IsStrikerOnLeft {
			currentState.BatsmanName = currentState.BatsmanLeft
		} else {
			currentState.BatsmanName = currentState.BatsmanRight
		}
		return
	}
	if currentState.BatsmanName == "" {
		currentState.BatsmanName = scoreboardBatsman
	}
}

// containsDigit checks if a string contains any numeric digit
func containsDigit(s string) bool {
	for _, char := range s {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

// cleanZoneName removes OCR noise like triangles or dots from the small zone text
func cleanZoneName(text string) string {
	text = strings.TrimSpace(text)
	textLower := strings.ToLower(text)

	// List of forbidden words that are part of HUD overlays
	forbidden := []string{
		"footwork", "shot choice", "timing", "shot ch", "shot",
		"foc", "foo", "choice", "ideal", "good", "early", "late",
		"nork timit", "ork", "tim", "work", "over", "run",
		"average", "wicket", "rate", "delivery", "speed", "remaining", "today",
		"overs", "runs", "balls", "minutes", "strike", "economy",
	}

	for _, word := range forbidden {
		if strings.Contains(textLower, word) {
			return ""
		}
	}

	// Remove common leading indicators
	text = regexp.MustCompile(`^([|>I]{0,2}>|[|▶]|\.|\s)+`).ReplaceAllString(text, "")

	// Final check: name should have some length and not be just noise
	if len(text) < 2 {
		return ""
	}

	// Reject if the text is primarily numeric (likely a score or balls)
	digitCount := 0
	for _, char := range text {
		if char >= '0' && char <= '9' {
			digitCount++
		}
	}
	if float64(digitCount)/float64(len(text)) > 0.5 {
		return ""
	}

	// Use existing corrections
	return correctPlayerName(strings.Title(strings.ToLower(text)))
}

func hasStrikerIndicator(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	// Added more variants of what the triangle might look like to OCR
	return strings.Contains(t, ">") ||
		strings.Contains(t, "|>") ||
		strings.Contains(t, ">>") ||
		strings.Contains(t, "▶") ||
		strings.Contains(t, "»") ||
		strings.Contains(t, "►") ||
		strings.HasPrefix(t, "v") || // Sometimes OCR sees the triangle as a 'v'
		strings.HasPrefix(t, "i") // Or a stray 'i' or '|'
}

func matchesPlayer(a, b string) bool {
	a = strings.TrimSpace(strings.ToLower(a))
	b = strings.TrimSpace(strings.ToLower(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// lastNameOf extracts the last word from a player name.
// Works for both full names ("Mitch Marsh" → "marsh") and
// abbreviated HUD names ("M. Marsh" → "marsh").
func lastNameOf(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Fields(name)
	return strings.ToLower(parts[len(parts)-1])
}

// lastNamesMatch returns true when both names share the same last name,
// enabling "Mitch Marsh" to match "M. Marsh" and vice versa.
func lastNamesMatch(a, b string) bool {
	la := lastNameOf(a)
	lb := lastNameOf(b)
	return la != "" && lb != "" && la == lb
}
