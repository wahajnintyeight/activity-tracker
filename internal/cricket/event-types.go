package cricket

// CricketEventType represents the type of cricket event detected
type CricketEventType string

const (
	EventTypeWicket        CricketEventType = "WICKET"
	EventTypeBoundarySix   CricketEventType = "BOUNDARY_SIX"
	EventTypeBoundaryFour  CricketEventType = "BOUNDARY_FOUR"
	EventTypeRuns          CricketEventType = "RUNS"
	EventTypeBatsmanArrive CricketEventType = "BATSMAN_ARRIVE"
	EventTypeBatsmanDepart CricketEventType = "BATSMAN_DEPART"
	EventTypeBowlerArrive  CricketEventType = "BOWLER_ARRIVE"
	EventTypeMilestone     CricketEventType = "MILESTONE"
	EventTypeTeamMilestone CricketEventType = "TEAM_MILESTONE"
	EventTypeChaseUpdate   CricketEventType = "CHASE_UPDATE"
	EventTypeOverComplete  CricketEventType = "OVER_COMPLETE"
	EventTypeInningsChange CricketEventType = "INNINGS_CHANGE"
	EventTypeMatchWon      CricketEventType = "MATCH_WON"
)

// String returns the string representation of the event type
func (e CricketEventType) String() string {
	return string(e)
}
