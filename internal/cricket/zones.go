package cricket

type Zone struct {
	Rect [2][2]int
	Side string
}

type ZoneProfile struct {
	Left   []Zone
	Middle []Zone
}

var zoneProfilesByGame = map[GameType]ZoneProfile{
	GameTypeC24: {
		Left: []Zone{
			{Rect: [2][2]int{{440, 635}, {75, 135}}, Side: "left"},
			{Rect: [2][2]int{{810, 1000}, {75, 135}}, Side: "right"},
		},
		Middle: []Zone{
			{Rect: [2][2]int{{235, 500}, {75, 135}}, Side: "left"},
			{Rect: [2][2]int{{575, 840}, {75, 135}}, Side: "right"},
		},
	},
	GameTypeC26: {
		Left: []Zone{
			{Rect: [2][2]int{{470, 670}, {75, 135}}, Side: "left"},
			{Rect: [2][2]int{{840, 1040}, {75, 135}}, Side: "right"},
		},
		Middle: []Zone{
			{Rect: [2][2]int{{55, 400}, {165, 200}}, Side: "left"},
			{Rect: [2][2]int{{55, 400}, {125, 160}}, Side: "right"}, 
		},
	},
}

func GetZones(gameType GameType, teamScorePosition string) []Zone {
	p, ok := zoneProfilesByGame[gameType]
	if !ok {
		p = zoneProfilesByGame[GameTypeC24]
	}

	switch teamScorePosition {
	case "left":
		return p.Left
	case "middle":
		return p.Middle
	default:
		zones := make([]Zone, 0, len(p.Left)+len(p.Middle))
		zones = append(zones, p.Left...)
		zones = append(zones, p.Middle...)
		return zones
	}
}
