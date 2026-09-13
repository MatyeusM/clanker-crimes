# Clanker Weather Ambience

A weather-reactive desktop overlay written in C, with Lua configuration,
built with [SDL3](https://wiki.libsdl.org/SDL3/FrontPage). It shades your
desktop background with subtle effects driven by the live weather at your
configured location: rain streaks when it rains, drifting snow when it
snows, a dark tint at night, cloud dimming, fog, wind streaks, lightning
flashes during thunderstorms, and cold/hot tints from the temperature.

## Features

- Transparent fullscreen overlay: borderless, always-on-top, non-focusable,
  with native click-through (Windows, macOS, X11)
- Rain, snow, wind, fog, storm flashes, night overlay, cloud dimming,
  temperature tint — each with its own intensity, color, count, and speed
- Lua configuration with sane defaults; every knob is optional
- Config chain: built-in defaults ← global file ← `./ambience.lua`
  (local wins, merged per-field, so partial files keep working)
- Keyless weather via [Open-Meteo](https://open-meteo.com/): place-name
  geocoding plus current forecast (no API key, no account)
- Degree-aware units: `metric` (°C, km/h) or `imperial` (°F, mph)
- Headless `--once` mode: prints config + weather + FX intensities, no window
- Small by design: ~2000 lines of C across files of at most ~240 lines

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin CMake and Ninja
(see `mise.toml`). You also need a C23 compiler and a few system packages.

**macOS / Linux:**

```sh
curl https://mise.run | sh
```

**Windows:**

```powershell
winget install jdx.mise
```

**System packages:**

```sh
# Debian/Ubuntu (X11 Shape headers enable click-through on X11)
sudo apt install build-essential libx11-dev libxext-dev

# macOS (Xcode command line tools are enough)
xcode-select --install
```

All third-party libraries (SDL3, yyjson, libcurl, Lua 5.5) are fetched and
built from source by CMake — nothing else to install.

## Getting started

```sh
# Install the pinned tools from mise.toml
mise install

# Configure the build
mise run configure

# Build
mise run build

# Run the overlay (quit with q / Esc)
mise run run

# Or call the binary directly
./build/clanker-weather-ambience --once
```

See `mise.toml` for the complete tool configuration and available tasks.

## Configuration

Copy `ambience.lua` next to the binary or edit it in place — every key is
optional, only set what you want to change:

```lua
location = "Nice, France"   -- or set latitude / longitude to skip geocoding
units = "metric"            -- "metric" | "imperial"
update_interval = 600       -- seconds between refetches (min 30)

rain = {
  count = 900,              -- max drops
  color = { r = 140, g = 170, b = 255, a = 140 },
  full_rate = 4.0,          -- mm/h that means "full intensity"
  intensity_scale = 1.0,    -- 0.5 for subtle, 1.5 for dramatic
}
night = {
  max_alpha = 110,          -- peak darkness overlay
}
```

The global file lives at:

- Linux: `$XDG_CONFIG_HOME/clanker/ambience.lua` or
  `~/.config/clanker/ambience.lua`
- macOS: `~/Library/Application Support/clanker/ambience.lua`
- Windows: `%APPDATA%\clanker\ambience.lua`

A `./ambience.lua` in the working directory overrides it per-field.

## CLI

```text
--once            print config + weather + intensities and exit (no window)
--config PATH     load this Lua file instead of the config chain
--location NAME   override the location for this run
--coords LAT LON  override with explicit coordinates
--help            usage
```

## Verifying

There is no test suite yet; `--once` is the smoke test — it exercises Lua
loading, geocoding, the forecast fetch, and every intensity mapping:

```sh
./build/clanker-weather-ambience --once
# fx intensities: rain=0.00 snow=0.00 wind=0.00 cloud=1.00 fog=0.00 thunder=0
```

`cmake --build build` is kept warning-free under `-Wall -Wextra -Wpedantic`.

## Project layout

```text
ambience.lua          # Example config (also works as ./ambience.lua override)
src/
├── main.c            # Entry point; CLI, SDL window, main loop, refetch timer
├── config.h          # Config structs and public loader API
├── config_defaults.c # Sane defaults for every knob
├── config_path.c     # Cross-platform global config path
├── config_lua.c      # Tolerant Lua value readers
├── config_apply.c    # Merges Lua globals into the live config
├── config_push.c     # Re-emits config as Lua globals (per-file merging)
├── config_load.c     # Load chain: defaults <- global <- local
├── http.h/.c         # Tiny GET helper over libcurl
├── weather.c         # Open-Meteo geocoding + forecast fetch
├── weather_eval.c    # Weather -> 0..1 FX intensities, labels, printing
├── fx.h              # Particle state and render API
├── fx.c              # Particle simulation (rain/snow/wind/fog/storm timer)
├── fx_render.c       # Fullscreen tints, particles, lightning flash
└── platform.h/.c     # Native click-through per OS
```

## License

MIT — do whatever you want with it, just don't blame me for your wallpaper.
