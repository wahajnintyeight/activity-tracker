package cricket

import (
	"fmt"
	"image"
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

// ProcessScoreWithVision analyzes OCR text and detects cricket events with the help of pixel scanning
func ProcessScoreWithVision(img *image.RGBA, currentText string, previous *MatchState) (*GameEvent, *MatchState) {
	currentText = strings.TrimSpace(currentText)

	if currentText == "" {
		return nil, previous
	}

	// Check if this is a milestone screen
	if isMilestoneScreen(currentText) {
		milestoneState := parseMilestoneDetails(currentText)

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

	// Check if this is a wicket dismissal screen
	if isWicketDismissalScreen(currentText) {
		dismissalState := parseDismissalDetails(currentText)

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

	// Check if this is a bowler stats screen
	if isBowlerStatsScreen(currentText) {
		bowlerStats := parseBowlerCareerStats(currentText)

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

	// Check if this is a batsman stats screen
	if isBatsmanStatsScreen(currentText) {
		careerStats := parseBatsmanCareerStats(currentText)

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

	// Process standard scoreboard
	if previous == nil {
		return nil, &MatchState{LastScore: currentText}
	}

	if currentText == previous.LastScore {
		return nil, previous
	}

	currentState := parseScoreText(currentText)
	if currentState == nil {
		return nil, previous
	}

	// Persist names from previous state if missing in current (e.g. during overlays)
	currentState.BatsmanLeft = previous.BatsmanLeft
	currentState.BatsmanRight = previous.BatsmanRight

	// If OCR found specific batsman names, update our Left/Right records
	updateBatsmanRecords(currentText, currentState)

	// VISUAL STRIKER DETECTION
	// Coordinates relative to your 1800x190 scoreboard crop.
	// Adjusting to target the small white triangle indicator.
	batsman1Zone := [2][2]int{{410, 595}, {45, 135}} // Left side striker arrow
	batsman2Zone := [2][2]int{{810, 1000}, {45, 135}} // Right side striker arrow

	// Make sure the image is large enough for the zones to avoid index out of bounds
	bounds := img.Bounds()
	if bounds.Dx() >= 795 && bounds.Dy() >= 100 {
		striker1 := DetectStriker(img, batsman1Zone[0], batsman1Zone[1], "left")
		striker2 := DetectStriker(img, batsman2Zone[0], batsman2Zone[1], "right")

		if striker1 {
			currentState.IsStrikerOnLeft = true
			fmt.Println("Vision: Striker detected on LEFT")
			if currentState.BatsmanLeft != "" {
				currentState.BatsmanName = currentState.BatsmanLeft
			}
		} else if striker2 {
			currentState.IsStrikerOnLeft = false
			fmt.Println("Vision: Striker detected on RIGHT")
			if currentState.BatsmanRight != "" {
				currentState.BatsmanName = currentState.BatsmanRight
			}
		} else {
			fmt.Println("Vision: No striker indicator found in zones")
		}
	}

	event := detectEvent(previous, currentState)
	currentState.LastScore = currentText

	return event, currentState
}
