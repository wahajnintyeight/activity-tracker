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
	EventTypeOverComplete  CricketEventType = "OVER_COMPLETE"
	EventTypeInningsChange CricketEventType = "INNINGS_CHANGE"
)

// String returns the string representation of the event type
func (e CricketEventType) String() string {
	return string(e)
}
