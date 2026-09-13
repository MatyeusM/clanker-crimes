// weather.h — Open-Meteo fetching (no API key) via libcurl + yyjson.
#pragma once

#include <stdbool.h>
#include "config.h"

typedef struct {
    bool valid;
    double temperature;     // according to units
    double apparent;
    double humidity;        // %
    double cloud_cover;     // %
    double wind_speed;      // km/h or mph
    double wind_dir;        // degrees
    double precipitation;   // mm
    double rain;            // mm/h
    double showers;         // mm/h
    double snowfall;        // cm/h
    double visibility;      // m
    int weather_code;       // WMO code
    int is_day;             // 0/1
    double latitude, longitude;
    char place[128];
    long long fetched_at; // unix seconds
} Weather;

// Fetch once (geocode location unless has_coords, then current forecast).
// Returns 0 on success. On failure returns nonzero and leaves `out` invalid.
int weather_fetch(const Config *cfg, Weather *out);

// Derived intensities 0..1 for the FX layer.
float weather_rain_intensity(const Weather *w, const Config *c);
float weather_snow_intensity(const Weather *w, const Config *c);
float weather_wind_intensity(const Weather *w, const Config *c);
float weather_cloud_factor(const Weather *w); // 0..1
float weather_fog_factor(const Weather *w);   // 0..1
bool weather_is_thunder(const Weather *w);
const char *weather_code_label(int code);

void weather_print(const Weather *w);
