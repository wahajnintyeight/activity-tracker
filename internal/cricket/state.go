package cricket

import (
	"fmt"
	"image"
	"regexp"
	"strings"
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
func ProcessScoreWithVision(img *image.RGBA, currentText string, previous *MatchState, ocr OCRClient, debug bool) (*GameEvent, *MatchState) {
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

	// 1. Check for SPECIAL SCREENS first (Wickets, Arrivals)

	// BATSMAN DEPARTING (Wicket Screen)
	if isWicketDismissalScreen(currentText) {
		dismissalState := parseDismissalDetails(currentText)

		// Logic: Identify who left and clear their slot
		departedName := strings.ToLower(dismissalState.BatsmanName)
		if strings.Contains(strings.ToLower(currentState.BatsmanLeft), departedName) || strings.Contains(departedName, strings.ToLower(currentState.BatsmanLeft)) {
			currentState.BatsmanLeft = ""
		} else if strings.Contains(strings.ToLower(currentState.BatsmanRight), departedName) || strings.Contains(departedName, strings.ToLower(currentState.BatsmanRight)) {
			currentState.BatsmanRight = ""
		}

		payload := fmt.Sprintf("Wicket! %s dismissed for %d. Score: %d/%d",
			dismissalState.BatsmanName, dismissalState.BatsmanRuns, dismissalState.Wickets, dismissalState.TotalRuns)

		event := &GameEvent{
			Type:      EventTypeBatsmanDepart,
			Payload:   payload,
			Raw:       currentText,
			MatchData: dismissalState,
		}
		currentState.LastScore = currentText
		return event, currentState
	}

	// BATSMAN ARRIVING (Career Stats Screen)
	if isBatsmanStatsScreen(currentText) {
		arrivalState := parseBatsmanCareerStats(currentText)

		// Logic: Fill the empty slot
		if currentState.BatsmanLeft == "" {
			currentState.BatsmanLeft = arrivalState.BatsmanName
		} else if currentState.BatsmanRight == "" && arrivalState.BatsmanName != currentState.BatsmanLeft {
			currentState.BatsmanRight = arrivalState.BatsmanName
		}

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
		return event, currentState
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
		return event, currentState
	}

	// 2. STANDARD SCOREBOARD PROCESSING
	if previous != nil && currentText == previous.LastScore {
		updateBatsmenAndStrikerFromZones(img, currentState, previous, "", ocr, debug)
		currentState.LastScore = currentText
		return nil, currentState
	}

	scoreboardState := parseScoreText(currentText)
	if scoreboardState == nil {
		updateBatsmenAndStrikerFromZones(img, currentState, previous, "", ocr, debug)
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

	updateBatsmenAndStrikerFromZones(img, currentState, previous, scoreboardState.BatsmanName, ocr, debug)

	event := detectEvent(previous, currentState)
	currentState.LastScore = currentText

	return event, currentState
}

func updateBatsmenAndStrikerFromZones(img *image.RGBA, currentState, previous *MatchState, scoreboardBatsman string, ocr OCRClient, debug bool) {
	if img == nil || currentState == nil || ocr == nil {
		return
	}

	// Targeted scoreboard zones: left and right batsman HUD strips.
	var zones = []struct {
		rect [2][2]int
		side string
	}{
		{rect: [2][2]int{{440, 635}, {75, 135}}, side: "left"},
		{rect: [2][2]int{{810, 1000}, {75, 135}}, side: "right"},
	}

	bounds := img.Bounds()
	if bounds.Dx() < 1000 || bounds.Dy() < 130 {
		return
	}

	strikerDetectedThisFrame := false
	for _, z := range zones {
		rect := image.Rect(z.rect[0][0], z.rect[1][0], z.rect[0][1], z.rect[1][1])
		sub := img.SubImage(rect).(*image.RGBA)
		zoneText, _ := ocr.ExtractText(sub)

		cleanName := cleanZoneName(zoneText)
		if cleanName != "" {
			if z.side == "left" && currentState.BatsmanLeft == "" {
				currentState.BatsmanLeft = cleanName
			} else if z.side == "right" && currentState.BatsmanRight == "" {
				currentState.BatsmanRight = cleanName
			}
		}

		if DetectStriker(sub, z.side, debug) || hasStrikerIndicator(zoneText) {
			currentState.IsStrikerOnLeft = (z.side == "left")
			strikerDetectedThisFrame = true
		}
	}

	if containsDigit(currentState.BatsmanLeft) {
		for _, z := range zones {
			if z.side != "left" {
				continue
			}
			rect := image.Rect(z.rect[0][0], z.rect[1][0], z.rect[0][1], z.rect[1][1])
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
			if z.side != "right" {
				continue
			}
			rect := image.Rect(z.rect[0][0], z.rect[1][0], z.rect[0][1], z.rect[1][1])
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
	return strings.Contains(t, ">") ||
		strings.Contains(t, "|>") ||
		strings.Contains(t, ">>") ||
		strings.Contains(t, "▶")
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

