# Screenshot Service

A Windows service that captures screenshots periodically and uploads them to an API.

## Project Structure

```
screenshot-service/
├── cmd/
│   └── service/          # Application entry point
│       └── main.go
├── internal/
│   ├── capture/          # Screenshot capture logic
│   │   └── capturer.go
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── service/          # Service orchestration
│   │   └── service.go
│   ├── uploader/         # API upload logic
│   │   └── uploader.go
│   └── window/           # Active window detection
│       └── window.go
├── .env                  # Configuration (create from .env.example)
├── .env.example          # Configuration template
├── build.bat             # Build script
└── go.mod
```

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Install RabbitMQ:
   - Download from: https://www.rabbitmq.com/download.html
   - Or use Docker: `docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management`
   - Management UI: http://localhost:15672 (guest/guest)

3. Configure the service:
```bash
copy .env.example .env
```
Edit `.env` and set:
   - `API_URL`: Your API endpoint (fallback)
   - `SCREENSHOT_INTERVAL`: Time between screenshots (e.g., `5m`, `30s`)
   - `JPEG_QUALITY`: JPEG quality (1-100, lower = smaller file)
   - `USE_RABBITMQ`: Enable RabbitMQ publishing (true/false)
   - `RABBITMQ_URL`: RabbitMQ connection string
   - `RABBITMQ_EXCHANGE`: Exchange name (must match worker-service config)
   - `RABBITMQ_ROUTING_KEY`: Routing key for messages

4. Build the executable:
```bash
build.bat
```

## Usage

### Option 1: Run Manually (No Service)

Just run it directly - stops when you close the terminal:

```cmd
screenshot-service.exe run
```

### Option 2: Install as Windows Service

Run as Administrator:

```cmd
screenshot-service.exe install
screenshot-service.exe start
```

### Service Management

```cmd
screenshot-service.exe stop
screenshot-service.exe restart
screenshot-service.exe uninstall
```

## Notes

- Service runs in the background without UI
- Logs are written to Windows Event Viewer
- Captures from primary display only
- Sends device name + timestamp with each screenshot
- Detects and sends active window info (title, process name, process ID)
- Only captures screenshots when user is active (not idle for 3+ minutes)
