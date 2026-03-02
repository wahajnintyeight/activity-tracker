package cricket

import (
	"regexp"
	"strconv"
	"strings"
)

// parseScoreText extracts runs, wickets, and overs from OCR text
func parseScoreText(text string) *MatchState {
	originalText := text
	state := &MatchState{}
	lines := strings.Split(text, "\n")

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
		bowlerWicketsRunsPattern := regexp.MustCompile(`([a-z]\.?\s+[a-z]+)\s+(\d+)-(\d+)`)
		if matches := bowlerWicketsRunsPattern.FindStringSubmatch(lineLower); len(matches) >= 4 {
			state.BowlerName = correctPlayerName(strings.Title(matches[1]))
			state.BowlerWickets, _ = strconv.Atoi(matches[2])
			state.BowlerRunsGiven, _ = strconv.Atoi(matches[3])
		}

		// Pattern 4b: Bowler name and economy (e.g., "j. archer 1:25")
		bowlerPattern := regexp.MustCompile(`([a-z]\.?\s+[a-z]+)\s+(\d+:\d+)`)
		if matches := bowlerPattern.FindStringSubmatch(lineLower); len(matches) >= 3 {
			if state.BowlerName == "" {
				state.BowlerName = correctPlayerName(strings.Title(matches[1]))
			}
			state.BowlerEconomy = matches[2]
		}

		// Pattern 5: Batsman stats - Name followed by runs and balls (e.g., "a. wasaie 0 11")
		batsmanStatsPattern := regexp.MustCompile(`([|>I]{0,2}>|[|▶])?\s*([a-z]\.?[a-z\s]+)\s+(\d+)\s+(\d+)`)
		if matches := batsmanStatsPattern.FindStringSubmatch(lineLower); len(matches) >= 5 {
			hasIndicator := matches[1] != ""
			name := correctPlayerName(strings.Title(strings.TrimSpace(matches[2])))

			if hasIndicator || state.BatsmanName == "" {
				state.BatsmanName = name
				state.BatsmanRuns, _ = strconv.Atoi(matches[3])
				state.BatsmanBalls, _ = strconv.Atoi(matches[4])
			}
		}

		// Pattern 5b: Batsman with > separator between two names (e.g., "m. labuschagne > c. green 0 0")
		batsmanSeparatorPattern := regexp.MustCompile(`([a-z]\.?[a-z\s]+)\s*>\s*([a-z]\.?[a-z\s]+)`)
		if matches := batsmanSeparatorPattern.FindStringSubmatch(lineLower); len(matches) >= 3 {
			state.BatsmanName = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}

		// Pattern 6: Player name with initial (e.g., "j. archer", "a. wasaie")
		playerNamePattern := regexp.MustCompile(`^([a-z]\.?\s+[a-z]+)$`)
		if matches := playerNamePattern.FindStringSubmatch(lineLower); len(matches) >= 2 {
			name := correctPlayerName(strings.Title(matches[1]))
			if state.BowlerName == "" && state.BatsmanName == "" {
				state.BatsmanName = name
			}
		}

		// Pattern 8: Wicket Summary (Specific to Cricket 24 "FALL OF WICKET" screen)
		if strings.Contains(lineLower, "fall of wicket") || strings.Contains(lineLower, "minutes balls") {
			potentialNamePattern := regexp.MustCompile(`^([a-z\s]{5,25})`)
			if nameMatches := potentialNamePattern.FindStringSubmatch(lineLower); len(nameMatches) >= 2 {
				name := strings.TrimSpace(nameMatches[1])
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

	if state.TotalRuns == 0 && state.Wickets == 0 {
		return nil
	}

	return state
}

func parseMilestoneDetails(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

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

	runsPattern := regexp.MustCompile(`(\d+)\*`)
	if matches := runsPattern.FindStringSubmatch(text); len(matches) >= 2 {
		runs, _ := strconv.Atoi(matches[1])
		state.BatsmanRuns = runs
		state.MilestoneRuns = runs
		state.MilestoneType = determineMilestoneType(runs)
	}

	ballsPattern := regexp.MustCompile(`balls\s+(\d+)`)
	if matches := ballsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanBalls, _ = strconv.Atoi(matches[1])
	}

	strikeRatePattern := regexp.MustCompile(`strike rate\s+(\d+\.?\d*)`)
	if matches := strikeRatePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanStrikeRate, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

func parseDismissalDetails(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	state.BatsmanName = extractDismissedBatsmanName(text)

	caughtPattern := regexp.MustCompile(`c\.?\s+([a-z\.\s]+?)\s+b\.?\s+([a-z\.\s]+?)(?:\s+\d+s|\s+balls|$)`)
	if matches := caughtPattern.FindStringSubmatch(textLower); len(matches) >= 3 {
		state.DismissalType = "caught"
		state.DismissalFielder = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		state.DismissalBowler = correctPlayerName(strings.Title(strings.TrimSpace(matches[2])))
	} else {
		bowledPattern := regexp.MustCompile(`b\.?\s+([a-z\.\s]+?)(?:\s+\d+s|\s+balls|$)`)
		if matches := bowledPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
			state.DismissalType = "bowled"
			state.DismissalBowler = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}
	}

	if strings.Contains(textLower, "run out") {
		state.DismissalType = "run out"
		runOutPattern := regexp.MustCompile(`run out\s*\(([a-z\.\s]+)\)`)
		if matches := runOutPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
			state.DismissalFielder = correctPlayerName(strings.Title(strings.TrimSpace(matches[1])))
		}
	}

	if strings.Contains(textLower, "lbw") {
		state.DismissalType = "lbw"
	}

	ballsPattern := regexp.MustCompile(`balls\s+(\d+)`)
	if matches := ballsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanBalls, _ = strconv.Atoi(matches[1])
	}

	strikeRatePattern := regexp.MustCompile(`strike rate\s+(\d+\.?\d*)`)
	if matches := strikeRatePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BatsmanStrikeRate, _ = strconv.ParseFloat(matches[1], 64)
	}

	fallOfWicketIdx := strings.Index(textLower, "fall of wicket")
	if fallOfWicketIdx > 0 {
		beforeFall := textLower[:fallOfWicketIdx]
		runsPattern := regexp.MustCompile(`(\d+)\s*$`)
		if matches := runsPattern.FindStringSubmatch(beforeFall); len(matches) >= 2 {
			state.BatsmanRuns, _ = strconv.Atoi(matches[1])
		}
	}

	fowPattern := regexp.MustCompile(`fall of wicket\s+(\d+)[/-](\d+)`)
	if matches := fowPattern.FindStringSubmatch(textLower); len(matches) >= 3 {
		state.Wickets, _ = strconv.Atoi(matches[1])
		state.TotalRuns, _ = strconv.Atoi(matches[2])
	}

	return state
}

func parseBowlerCareerStats(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	state.BowlerName = extractBowlerNameFromStats(text)

	matchesPattern := regexp.MustCompile(`matches\s+(\d+)`)
	if matches := matchesPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerMatches, _ = strconv.Atoi(matches[1])
	}

	wicketsPattern := regexp.MustCompile(`wickets\s+(\d+)`)
	if matches := wicketsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BowlerWickets, _ = strconv.Atoi(matches[1])
	}

	runsPattern := regexp.MustCompile(`runs\s+(\d+)`)
	if matches := runsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.BowlerRunsGiven, _ = strconv.Atoi(matches[1])
	}

	averagePattern := regexp.MustCompile(`average\s+(\d+\.?\d*)`)
	if matches := averagePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerAverage, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

func parseBatsmanCareerStats(text string) *MatchState {
	state := &MatchState{}
	textLower := strings.ToLower(text)

	state.BatsmanName = extractBatsmanNameFromStats(text)

	matchesPattern := regexp.MustCompile(`matches\s+(\d+)`)
	if matches := matchesPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerMatches, _ = strconv.Atoi(matches[1])
	}

	runsPattern := regexp.MustCompile(`runs\s+(\d+)`)
	if matches := runsPattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerRuns, _ = strconv.Atoi(matches[1])
	}

	averagePattern := regexp.MustCompile(`average\s+(\d+\.?\d*)`)
	if matches := averagePattern.FindStringSubmatch(textLower); len(matches) >= 2 {
		state.CareerAverage, _ = strconv.ParseFloat(matches[1], 64)
	}

	return state
}

func correctPlayerName(name string) string {
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

	if corrected, ok := corrections[name]; ok {
		return corrected
	}

	nameLower := strings.ToLower(name)
	for wrong, right := range corrections {
		if strings.Contains(nameLower, strings.ToLower(wrong)) {
			return strings.Replace(name, wrong, right, 1)
		}
	}

	return name
}

func extractDismissedBatsmanName(text string) string {
	textLower := strings.ToLower(text)

	minutesIdx := strings.Index(textLower, "minutes")
	if minutesIdx == -1 {
		runOutIdx := strings.Index(textLower, "run out")
		if runOutIdx != -1 {
			minutesIdx = runOutIdx
		}
	}

	if minutesIdx == -1 {
		return "Unknown Batsman"
	}

	beforeKeyword := strings.TrimSpace(text[:minutesIdx])
	lines := strings.Split(beforeKeyword, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		if strings.Contains(lineLower, " c. ") ||
			strings.Contains(lineLower, " b. ") ||
			strings.Contains(lineLower, "lbw") ||
			strings.Contains(lineLower, "fall of wicket") {
			continue
		}

		if len(line) > 2 && len(line) < 50 {
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

func extractBowlerNameFromStats(text string) string {
	textLower := strings.ToLower(text)

	matchesIdx := strings.Index(textLower, "matches")
	if matchesIdx == -1 {
		return "Unknown Bowler"
	}

	beforeMatches := strings.TrimSpace(text[:matchesIdx])
	lines := strings.Split(beforeMatches, "\n")
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
				return correctPlayerName(strings.Title(strings.ToLower(line)))
			}
		}
	}

	return "Unknown Bowler"
}

func extractBatsmanNameFromStats(text string) string {
	return extractBowlerNameFromStats(text) // Logic is identical
}

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
		return strconv.Itoa(runs) + " runs"
	}
}

// updateBatsmanRecords updates the Left/Right player names in state based on OCR text
func updateBatsmanRecords(text string, state *MatchState) {
	textLower := strings.ToLower(text)
	lines := strings.Split(textLower, "\n")

	// Helper regex to find name + runs + balls
	pattern := regexp.MustCompile(`([a-z]\.?[a-z\s]+)\s+\d+\s+\d+`)

	for _, line := range lines {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			name := correctPlayerName(strings.Title(strings.TrimSpace(match[1])))

			// Simple logic: if OCR found two names, assign them based on order
			// This assumes standard HUD layout: [Left Name] ... [Right Name]
			if state.BatsmanLeft == "" {
				state.BatsmanLeft = name
			} else if state.BatsmanRight == "" && name != state.BatsmanLeft {
				state.BatsmanRight = name
			}
		}
	}
}
