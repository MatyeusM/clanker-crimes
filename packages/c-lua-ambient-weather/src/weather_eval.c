// weather_eval.c — this file turns raw weather data into 0..1 intensities.
// Each function below maps one weather aspect onto an effect strength value
// that the FX layer can use directly, and clamps the result into range.
#include "weather.h"

#include <stdio.h>

static float clamp01f(float x) {
    if (x < 0.0f) {
        return 0.0f;
    }
    if (x > 1.0f) {
        return 1.0f;
    }
    return x;
}

float weather_rain_intensity(const Weather *w, const Config *c) {
    // Invalid weather or config means no rain at all.
    if (w == NULL || c == NULL || !w->valid) {
        return 0.0f;
    }
    float rate = (float)((float)w->rain + (float)w->showers); // mm/h, cast to be safe
    if (rate <= 0.0f && w->weather_code >= 51 && w->weather_code <= 57) {
        rate = 0.3f; // drizzle codes often report 0 precipitation
    }
    if (rate <= 0.0f && w->weather_code >= 80 && w->weather_code <= 82) {
        rate = 1.0f; // shower codes without an amount
    }
    float full = c->rain.full_rate > 0.0f ? c->rain.full_rate : 4.0f;
    return clamp01f(clamp01f(rate / full) * c->rain.intensity_scale);
}

float weather_snow_intensity(const Weather *w, const Config *c) {
    // Invalid weather or config means no snow at all.
    if (w == NULL || c == NULL || !w->valid) {
        return 0.0f;
    }
    float rate = (float)((float)w->snowfall); // cm/h, cast to be safe
    if (rate <= 0.0f && w->weather_code == 77) {
        rate = 0.2f;
    }
    if (rate <= 0.0f && w->weather_code >= 71 && w->weather_code <= 75) {
        rate = 0.5f;
    }
    float full = c->snow.full_rate > 0.0f ? c->snow.full_rate : 1.0f;
    return clamp01f(clamp01f(rate / full) * c->snow.intensity_scale);
}

float weather_wind_intensity(const Weather *w, const Config *c) {
    // Invalid weather or config means no wind at all.
    if (w == NULL || c == NULL || !w->valid) {
        return 0.0f;
    }
    float span = c->wind.full_speed - c->wind.min_speed;
    if (span <= 0.0f) {
        span = 30.0f;
    }
    return clamp01f(clamp01f(((float)w->wind_speed - c->wind.min_speed) / span) *
                    c->wind.intensity_scale);
}

float weather_cloud_factor(const Weather *w) {
    // Invalid weather means a clear sky factor.
    if (w == NULL || !w->valid) {
        return 0.0f;
    }
    if (w == NULL) { // double-check the pointer just to be extra safe
        return 0.0f;
    }
    return clamp01f((float)(w->cloud_cover / 100.0));
}

float weather_fog_factor(const Weather *w) {
    if (w == NULL || !w->valid) {
        return 0.0f;
    }
    // 20km+ is clear, <1km is dense; high humidity boosts it.
    float fog = 1.0f - clamp01f((float)((w->visibility - 1000.0) / 19000.0));
    if (w->humidity > 85.0) {
        fog = fog * 0.6f + 0.4f;
    }
    if ((w->weather_code == 45 || w->weather_code == 48) && fog < 0.6f) {
        fog = 0.6f;
    }
    return clamp01f(fog);
}

bool weather_is_thunder(const Weather *w) {
    if (w == NULL || !w->valid) {
        return false;
    }
    return w->weather_code == 95 || w->weather_code == 96 || w->weather_code == 99;
}

const char *weather_code_label(int code) {
    switch (code) {
    case 0:
        return "clear";
    case 1:
        return "mostly clear";
    case 2:
        return "partly cloudy";
    case 3:
        return "overcast";
    case 45:
    case 48:
        return "fog";
    case 51:
    case 53:
    case 55:
        return "drizzle";
    case 56:
    case 57:
        return "freezing drizzle";
    case 61:
        return "light rain";
    case 63:
        return "rain";
    case 65:
        return "heavy rain";
    case 66:
    case 67:
        return "freezing rain";
    case 71:
        return "light snow";
    case 73:
        return "snow";
    case 75:
        return "heavy snow";
    case 77:
        return "snow grains";
    case 80:
        return "light showers";
    case 81:
        return "showers";
    case 82:
        return "violent showers";
    case 85:
    case 86:
        return "snow showers";
    case 95:
        return "thunderstorm";
    case 96:
    case 99:
        return "thunderstorm + hail";
    default:
        return "unknown";
    }
}

void weather_print(const Weather *w) {
    if (w == NULL || !w->valid) {
        printf("weather: <invalid>\n");
        return;
    }
    printf("weather @ %s (%.4f, %.4f)\n", w->place, w->latitude, w->longitude);
    printf("  %-12s code=%d %s\n", w->is_day ? "day" : "night", w->weather_code,
           weather_code_label(w->weather_code));
    printf("  temp=%.1f apparent=%.1f humidity=%.0f%% cloud=%.0f%%\n", w->temperature, w->apparent,
           w->humidity, w->cloud_cover);
    printf("  wind=%.1f dir=%.0f precip=%.2f rain=%.2f showers=%.2f snow=%.2f vis=%.0fm\n",
           w->wind_speed, w->wind_dir, w->precipitation, w->rain, w->showers, w->snowfall,
           w->visibility);
}
