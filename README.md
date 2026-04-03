# Activity Tracker

A Windows service that captures screenshots and tracks cricket game events.

## Features

### Activity Tracker Mode (Default)
- Captures screenshots at configurable intervals
- Publishes to RabbitMQ for processing
- Runs as Windows service or standalone

### Cricket Tracker Mode
- Monitors Cricket 24 game in real-time
- Two OCR modes:
  - **Local OCR**: Uses Windows Native OCR (same engine as Snipping Tool)
  - **LLM OCR**: Sends scoreboard images to queue for server-side LLM analysis
- Detects boundaries (4s, 6s), wickets, and runs
- Publishes events to RabbitMQ for LLM commentary generation
- No external OCR dependencies required (built into Windows 10/11)

## Installation

### Prerequisites
- Go 1.23+
- Windows 10/11 (for Windows Native OCR in cricket tracking)
- RabbitMQ server

### Install Dependencies
```bash
go mod download
```

### Build
```bash
.\scripts\build.bat
```

## Usage

### Activity Tracker (Screenshot Service)
```bash
# Run manually
.\output\service.exe

# Install as Windows service
.\scripts\install-startup.bat

# Uninstall service
.\scripts\uninstall-startup.bat
```

### Cricket Tracker
```bash
# Run cricket tracker
.\output\service.exe --type cricket-tracker
```

## Configuration

Edit `.env` file:

### Activity Tracker Settings
```env
SCREENSHOT_INTERVAL=5m
JPEG_QUALITY=50
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=worker-service-exchange
RABBITMQ_ROUTING_KEY=process-screenshot
```

### Cricket Tracker Settings
```env
CRICKET_SCAN_INTERVAL=2s
CRICKET_RABBITMQ_EXCHANGE=worker-service-exchange
CRICKET_RABBITMQ_ROUTING_KEY=cricket-event

# Adjust these based on your screen resolution
CRICKET_SCOREBOARD_X=20
CRICKET_SCOREBOARD_Y=950
CRICKET_SCOREBOARD_WIDTH=300
CRICKET_SCOREBOARD_HEIGHT=80

# Process names to monitor
CRICKET_PROCESS_NAMES=Cricket24.exe,cricket.exe,Cricket 24.exe
```

## Cricket Tracker Setup

1. **Find Scoreboard Coordinates**
   - Launch Cricket 24
   - Note the scoreboard position (usually bottom-left or bottom-center)
   - Measure X, Y, Width, Height in pixels
   - Update `.env` with these values

2. **Test OCR**
   - Run cricket tracker: `.\output\service.exe --type cricket-tracker`
   - Play a match
   - Check logs for detected events

3. **Adjust Settings**
   - If OCR is inaccurate, adjust scoreboard coordinates
   - Modify `CRICKET_SCAN_INTERVAL` for performance (default: 2s)

## Event Types

Cricket tracker detects:
- `BOUNDARY_FOUR` - Four runs scored
- `BOUNDARY_SIX` - Six runs scored
- `WICKET` - Wicket fallen
- `RUNS` - Regular runs (1, 2, 3)
- `BATSMAN_ARRIVE` - New batsman enters with career stats (matches, runs, average)
- `BATSMAN_DEPART` - Batsman dismissed with full details (bowler, fielder, runs, balls, strike rate, dismissal type)
- `OVER_COMPLETE` - Over completed
- `INNINGS_CHANGE` - Innings change

Events are published to RabbitMQ with detailed information:

**BATSMAN_DEPART Example:**
```json
{
  "type": "BATSMAN_DEPART",
  "payload": "Batsman dismissed: Marnus Labuschagne scored 13 runs off 11 balls (SR: 118.2) | Dismissal: caught (c. D. Warner) b. S. Afridi | Score: 7/51",
  "raw": "MARNI-JS LABUSCHAGNE MINUTES C. D WARNER b. S. AFRIDI 6s BALLS 11 STRIKE RATE 118.2 13 FALL OF WICKET 7/51",
  "match_data": {
    "batsman_name": "Marnus Labuschagne",
    "batsman_runs": 13,
    "batsman_balls": 11,
    "batsman_strike_rate": 118.2,
    "dismissal_type": "caught",
    "dismissal_fielder": "D. Warner",
    "dismissal_bowler": "S. Afridi",
    "wickets": 7,
    "total_runs": 51
  }
}
```

**BATSMAN_ARRIVE Example:**
```json
{
  "type": "BATSMAN_ARRIVE",
  "payload": "New batsman: Jasprit Bumrah | Career: 195 matches, 101 runs, avg 7.77",
  "raw": "JASPRIT BUMRAH MATCHES 195 RIGHT HAND BAT HUNDREDS RUNS 101 AVERAGE 7.77 STRIKE RATE 81.5 FIFTIES HIGH SCORE",
  "match_data": {
    "batsman_name": "Jasprit Bumrah",
    "career_matches": 195,
    "career_runs": 101,
    "career_average": 7.77
  }
}
```

**Boundary events include bowler stats:**
```json
{
  "type": "BOUNDARY_SIX",
  "payload": "SIX! Score: 123/4 (Overs: 15.3) | Bowler: S. Thakur 0-30 (4.4) | Speed: 129.4 km/h",
  "raw": "123/4 (15.3)",
  "match_data": {
    "total_runs": 123,
    "wickets": 4,
    "overs": 15.3,
    "bowler_name": "S. Thakur",
    "bowler_wickets": 0,
    "bowler_runs_given": 30,
    "bowler_overs": 4.4,
    "delivery_speed": "129.4 km/h"
  }
}
```

## Architecture

```
activity-tracker (captures frames)
    ↓
RabbitMQ (cricket-event topic)
    ↓
project-phoenix-v2 worker-service (processes events)
    ↓
LLM Service (generates commentary)
    ↓
Discord (publishes colorful commentary)
```

## Troubleshooting

### OCR Not Working
- Ensure Tesseract is installed and in PATH
- Verify scoreboard coordinates are correct
- Check if Cricket 24 is in foreground

### No Events Detected
- Verify process name matches (check Task Manager)
- Adjust `CRICKET_PROCESS_NAMES` in `.env`
- Check RabbitMQ connection

### Performance Issues
- Increase `CRICKET_SCAN_INTERVAL` (e.g., 3s or 5s)
- Reduce scoreboard capture area size
