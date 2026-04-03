package discord

import (
	"fmt"
	"time"

	"github.com/tropicalshadow/rich-go/client"
)

// PresenceInfo holds data for Discord Rich Presence
type PresenceInfo struct {
	Details    string
	State      string
	LargeImage string
	LargeText  string
	SmallImage string
	SmallText  string
	StartTime  *time.Time
	HideTime   bool // If true, don't show the elapsed time in Discord
}

// UpdatePresence handles the actual IPC call to Discord
func (d *DiscordClient) UpdatePresence(info PresenceInfo) error {
	if d.appID == "" {
		return nil
	}

	if !d.client.IsLogged() {
		if err := d.client.Login(d.appID); err != nil {
			return err
		}
	}

	activity := client.Activity{
		Details:    info.Details,
		State:      info.State,
		LargeImage: info.LargeImage,
		LargeText:  info.LargeText,
		SmallImage: info.SmallImage,
		SmallText:  info.SmallText,
	}

	if !info.HideTime {
		if info.StartTime != nil {
			activity.Timestamps = &client.Timestamps{
				Start: info.StartTime,
			}
		} else {
			now := time.Now()
			activity.Timestamps = &client.Timestamps{
				Start: &now,
			}
		}
	}

	_, err := d.client.SetActivity(activity)
	return err
}

// FormatActivityPresence creates a PresenceInfo for general activity
func FormatActivityPresence(processName, title string) PresenceInfo {
	return PresenceInfo{
		Details:    fmt.Sprintf("Using: %s", processName),
		State:      title,
		LargeImage: "app_icon", // Placeholder for actual icon logic if added
		LargeText:  processName,
	}
}

// FormatCricketPresence creates a PresenceInfo for cricket tracking
func FormatCricketPresence(gameName, teamA, teamB, runs, wickets, overs, striker string, target, needRuns, needBalls int, lastEvent string) PresenceInfo {
	// Line 1: In a match: TEAM A vs TEAM B
	details := "In a match"
	if teamA != "" && teamB != "" {
		details = fmt.Sprintf("In a match: %s v %s", teamA, teamB)
	}

	// Line 2: Score & Striker / Chase / Event (Optimized for mobile)
	// Priority 1: Important Event (WICKET!, FOUR!, etc.)
	// Priority 2: Chase (N: 10r from 12b)
	// Priority 3: Standard (Score (Ov) - Striker*)
	
	state := fmt.Sprintf("%s/%s (%s) - %s*", runs, wickets, overs, striker)

	if lastEvent != "" {
		// Display event message prominently
		state = fmt.Sprintf("%s | %s/%s (%s)", lastEvent, runs, wickets, overs)
	} else if needRuns > 0 && needBalls > 0 {
		// New chase format: N: <RUNS>r from <BALLS>b
		state = fmt.Sprintf("%s/%s (%s) | N: %dr from %db", runs, wickets, overs, needRuns, needBalls)
	} else if target > 0 {
		state = fmt.Sprintf("%s/%s (%s) | T:%d", runs, wickets, overs, target)
	}

	// Truncate to avoid Discord cutting off text (approx 128 chars, but mobile is tighter)
	if len(details) > 64 {
		details = details[:61] + "..."
	}
	if len(state) > 64 {
		state = state[:61] + "..."
	}

	return PresenceInfo{
		Details:    details,
		State:      state,
		LargeImage: "cricket_icon",
		LargeText:  gameName,
		HideTime:   true, // Hide elapsed time as requested
	}
}
