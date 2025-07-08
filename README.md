# LAMZU Polling Rate Auto-Switch (Go)

A Go-based automator for LAMZU mice that automatically adjusts polling rate based on running games.

## Features

- ✅ Single executable (.exe) with no dependencies
- ✅ Low resource usage (native Go)
- ✅ No GUI - CLI/Service only
- ✅ YAML configuration file
- ✅ Windows service support
- ✅ Native LAMZU Maya X 8K HID protocol
- 🆕 **Native Windows HID API**: Direct implementation using hid.dll and setupapi.dll
- 🆕 **Better device detection**: Uses SetupDi APIs for maximum reliability

## Installation

1. Download the `lamzu-automator.exe` executable
2. Run `build.bat` to compile from source (optional)
3. Configure your games in `config.yaml`
4. Run as administrator

## Usage

### Interactive Mode
```bash
# Run normally
lamzu-automator.exe

# Run with custom configuration file
lamzu-automator.exe -c my-config.yaml

# Run with verbose output
lamzu-automator.exe -v
```

### Daemon/Service Mode
```bash
# Run as daemon (background)
lamzu-automator.exe -d

# Install as Windows service
install.bat

# Remove Windows service
uninstall.bat
```

### Manual Commands
```bash
# Set polling rate manually
lamzu-automator.exe set 2000

# List available polling rates
lamzu-automator.exe list

# Debug and test device connection
lamzu-automator.exe debug

# Help
lamzu-automator.exe --help
```

## Configuration

Edit the `config.yaml` file:

```yaml
default_polling_rate: 1000  # Default polling rate (desktop)
game_polling_rate: 2000     # Polling rate for games
check_interval: 2s          # Check interval
games:                      # List of games (processes)
  - HuntGame.exe
  - DuneSandbox-Wi.exe
  - eldenring.exe
  - cs2.exe
  - valorant.exe
  - ApexLegends.exe
```

## Requirements

- Windows 10/11
- LAMZU Maya X 8K mouse
- Run as Administrator (required for HID access)

## Advantages vs Node.js Version

- **Size**: ~5MB vs ~80MB (16x smaller)
- **Memory**: ~10MB vs ~50MB (5x less)
- **Startup**: Instant vs ~2 seconds
- **Dependencies**: Zero vs Node.js + Electron
- **Security**: Native executable vs JavaScript

## Building

```bash
# Install Go 1.21+
go mod tidy
go build -ldflags="-s -w" -o lamzu-automator.exe .
```

## Native HID Implementation

The application uses Windows native APIs directly for maximum reliability:

**Native Windows API**: Uses `hid.dll`, `setupapi.dll` directly
- More reliable device discovery
- Uses `HidD_GetHidGuid`, `SetupDiGetClassDevs`
- Filters by interface (interface 2 for LAMZU)
- Commands via `HidD_SetFeature` for feature reports
- Better Windows system integration

Run with `-v` to see device discovery details:
```bash
lamzu-automator.exe debug -v
```

## Troubleshooting

**"device not found" error**:
- Run as Administrator
- Check if mouse is connected
- Confirm it's a LAMZU Maya X 8K
- Test: `lamzu-automator.exe debug -v`

**Polling rate doesn't change**:
- Restart the mouse (disconnect/reconnect)
- Check if other software is controlling the mouse
- Confirm the game process is in the configuration list

**Advanced debugging**:
```bash
# Debug with verbose output
lamzu-automator.exe debug -v
```