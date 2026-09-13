// config.h — Lua-driven configuration with sane defaults.
//
// Load order (later overrides earlier, merged per-field):
//   1. built-in defaults
//   2. global config  ($XDG_CONFIG_HOME/clanker/ambience.lua,
//                      ~/.config/clanker/ambience.lua,
//                      ~/Library/Application Support/clanker/ambience.lua on macOS,
//                      %APPDATA%\clanker\ambience.lua on Windows)
//   3. local config   (./ambience.lua in the working directory)
#pragma once

#include <stdbool.h>
#include <stdint.h>

typedef struct {
    uint8_t r, g, b, a;
} Rgba;

typedef struct {
    bool enabled;
    Rgba color;
    uint8_t max_alpha; // overlay peak strength
} OverlayCfg; // night, temperature tint, clouds

typedef struct {
    bool enabled;
    Rgba color;
    uint8_t max_alpha;
    int count;      // fog puffs
    float speed;    // px/s drift
} FogCfg;

typedef struct {
    bool enabled;
    Rgba color;
    int count;          // max drops
    float length;       // px
    float speed;        // px/s fall speed
    float full_rate;    // mm/h that yields intensity 1.0
    float intensity_scale;
} RainCfg;

typedef struct {
    bool enabled;
    Rgba color;
    int count;          // max flakes
    float size_min, size_max; // px radius
    float speed;        // px/s fall speed
    float drift;        // px/s horizontal sway amplitude
    float full_rate;    // cm/h that yields intensity 1.0
    float intensity_scale;
} SnowCfg;

typedef struct {
    bool enabled;
    Rgba color;
    int count;          // max streaks
    float speed_scale;  // multiplier on wind speed
    float min_speed;    // km/h where streaks start appearing
    float full_speed;   // km/h that yields intensity 1.0
    float streak_length;// px
    float intensity_scale;
} WindCfg;

typedef struct {
    bool enabled;
    Rgba color;
    uint8_t flash_alpha;
    float bolts_per_min; // average lightning rate during thunderstorm
} StormCfg;

typedef struct {
    bool enabled; // master switch for the whole temperature tint
    float cold_threshold; // deg C (or F if units=imperial)
    float hot_threshold;
    Rgba cold_color;
    Rgba hot_color;
    uint8_t max_alpha;
} TempCfg;

typedef struct {
    char location[128]; // place name for geocoding, e.g. "Nice, France"
    double latitude;    // optional override
    double longitude;   // optional override
    bool has_coords;    // true when lat/lon override is set
    char units[16];     // "metric" | "imperial"
    int update_interval; // seconds between weather refetches

    struct {
        float opacity;       // 0..1 window opacity hint
        bool click_through;  // mouse passthrough
        bool always_on_top;
    } window;

    OverlayCfg night;
    TempCfg temperature;
    OverlayCfg clouds;
    FogCfg fog;
    RainCfg rain;
    SnowCfg snow;
    WindCfg wind;
    StormCfg storm;
} Config;

void config_defaults(Config *c);
// Loads defaults, then global file, then local file. Missing files are fine.
// Returns 0 on success, nonzero if a present file had a Lua syntax error.
int config_load(Config *c);
// Resolve a single explicit config file path (used by --config).
int config_load_file(Config *c, const char *path);
// Paths used (for logging). out may be NULL.
void config_global_path(char *out, unsigned long long cap);
void config_print(const Config *c);
