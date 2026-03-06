---
description: How to run the Cricket Activity Tracker
---

# Running the Cricket Activity Tracker

## Prerequisites

1. **RabbitMQ** is running (required for event publishing)
2. **Tesseract OCR** is installed (required for text recognition)
3. **Cricket 24 game** is running (the game being tracked)
4. **`.env` file** is configured with correct scoreboard coordinates

## Quick Start

// turbo
1. Build the application
   ```
   .\build.bat
   ```

2. Run the cricket tracker
   ```
   .\run-cricket-tracker.bat
   ```

## Alternative: Run as Service

// turbo
1. Build and install as Windows service
   ```
   .\build.bat
   .\install-service.bat
   ```

2. Or use the startup installer (runs on Windows login)
   ```
   .\install-startup.bat
   ```

## Debug Mode

To enable debug zone images for troubleshooting striker detection:

1. Set `DEBUG_ZONES=true` in your `.env` file
2. Debug images will be saved to `debug_zones/` folder

## Troubleshooting

- **"OCR returned empty"**: Check scoreboard coordinates in `.env` file
- **service.exe not found**: Run `build.bat` first
- **Connection errors**: Ensure RabbitMQ is running on the configured URL
