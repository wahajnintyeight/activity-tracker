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

		event := &GameEvent{
			Type:      EventTypeBatsmanArrive,
			Payload:   fmt.Sprintf("New batsman in: %s", arrivalState.BatsmanName),
			Raw:       currentText,
			MatchData: arrivalState,
		}
		currentState.LastScore = currentText
		return event, currentState
	}

	// 2. STANDARD SCOREBOARD PROCESSING
	if previous != nil && currentText == previous.LastScore {
		return nil, previous
	}

	scoreboardState := parseScoreText(currentText)
	if scoreboardState == nil {
		return nil, previous
	}

	// Update score-related fields
	currentState.TotalRuns = scoreboardState.TotalRuns
	currentState.Wickets = scoreboardState.Wickets
	currentState.Overs = scoreboardState.Overs
	currentState.BowlerName = scoreboardState.BowlerName
	currentState.DeliverySpeed = scoreboardState.DeliverySpeed

	// TARGETED OCR ON ZONES (Used to fill missing names or confirm)
	var zones = []struct {
		rect [2][2]int
		side string
	}{
		{rect: [2][2]int{{400, 595}, {85, 130}}, side: "left"},
		{rect: [2][2]int{{810, 1000}, {85, 130}}, side: "right"},
		// {rect: [2][2]int{{240, 500}, {85, 130}}, side: "left"},
		// {rect: [2][2]int{{580, 840}, {85, 130}}, side: "right"},
	}

	bounds := img.Bounds()
	if bounds.Dx() >= 1000 && bounds.Dy() >= 130 {
		for _, z := range zones {
			// Only OCR if we really need to (slot empty) or to verify
			rect := image.Rect(z.rect[0][0], z.rect[1][0], z.rect[0][1], z.rect[1][1])
			sub := img.SubImage(rect).(*image.RGBA)
			name, _ := ocr.ExtractText(sub)

			cleanName := cleanZoneName(name)
			if cleanName != "" {
				// If slot is empty, fill it immediately
				if z.side == "left" && currentState.BatsmanLeft == "" {
					currentState.BatsmanLeft = cleanName
				} else if z.side == "right" && currentState.BatsmanRight == "" {
					currentState.BatsmanRight = cleanName
				}
			}

			// VISUAL STRIKER DETECTION (Always check for arrow, even in number zones)
			if DetectStriker(sub, z.side, debug) {
				if z.side == "left" {
					currentState.IsStrikerOnLeft = true
				} else {
					currentState.IsStrikerOnLeft = false
				}
			}
		}

		// Update active batsman name after all zones processed
		if currentState.IsStrikerOnLeft {
			currentState.BatsmanName = currentState.BatsmanLeft
		} else {
			currentState.BatsmanName = currentState.BatsmanRight
		}
	}

	event := detectEvent(previous, currentState)
	currentState.LastScore = currentText

	return event, currentState
}

// cleanZoneName removes OCR noise like triangles or dots from the small zone text
func cleanZoneName(text string) string {
	text = strings.TrimSpace(text)
	textLower := strings.ToLower(text)

	// List of forbidden words that are part of HUD overlays
	forbidden := []string{
		"footwork", "shot choice", "timing", "shot ch", "shot",
		"foc", "foo", "choice", "ideal", "good", "early", "late",
		"nork timit", "ork", "tim", "work","over","run",
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
