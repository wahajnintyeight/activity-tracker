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
func FormatCricketPresence(gameName, teamA, teamB, runs, wickets, overs, striker string, target, needRuns, needBalls int) PresenceInfo {
	details := fmt.Sprintf("%s vs %s", teamA, teamB)
	if teamA == "" || teamB == "" {
		details = gameName
	}

	state := fmt.Sprintf("%s/%s (%s ov) - %s*", runs, wickets, overs, striker)
	if target > 0 {
		state = fmt.Sprintf("%s/%s (%s ov) | T:%d - %s*", runs, wickets, overs, target, striker)
	}
	if needRuns > 0 && needBalls > 0 {
		state = fmt.Sprintf("%s | Need %d from %d", state, needRuns, needBalls)
	}

	return PresenceInfo{
		Details:    details,
		State:      state,
		LargeImage: "cricket_icon",
		LargeText:  gameName,
	}
}
