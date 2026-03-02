package enum

type TrackerType int

const (
	// Define constants for each tracker type
	ActivityTracker TrackerType = iota
	CricketTracker
)

// String provides the string representation of the TrackerType
func (tt TrackerType) String() string {
	return [...]string{"ActivityTracker", "CricketTracker"}[tt]
}
