// fx.h — particle overlays (rain/snow/wind/fog) + fullscreen tints.
#pragma once

#include <SDL3/SDL.h>
#include "config.h"
#include "weather.h"

#define FX_MAX_RAIN 4000
#define FX_MAX_SNOW 4000
#define FX_MAX_WIND 1000
#define FX_MAX_FOG 256

typedef struct {
    float x, y;     // px
    float speed;    // fall multiplier 0.7..1.3
    float len;      // px jitter
} RainDrop;

typedef struct {
    float x, y;
    float r;        // radius px
    float speed;    // fall multiplier
    float phase;    // sway phase
    float sway;     // sway speed
} SnowFlake;

typedef struct {
    float x, y;
    float speed;    // px/s
    float len;      // px
} WindStreak;

typedef struct {
    float x, y;
    float r;        // radius px
    float speed;    // px/s horizontal drift
    float alpha;    // 0..1 per-puff variance
} FogPuff;

typedef struct {
    RainDrop rain[FX_MAX_RAIN];
    SnowFlake snow[FX_MAX_SNOW];
    WindStreak wind[FX_MAX_WIND];
    FogPuff fog[FX_MAX_FOG];
    int w, h; // last known viewport
    float storm_flash; // 0..1 current flash brightness (decays)
    float storm_timer; // seconds until next bolt
    unsigned long long rng;
} Fx;

void fx_init(Fx *fx, int w, int h);
void fx_resize(Fx *fx, int w, int h);

// Active particle counts for the current weather (0..configured max).
int fx_active_rain(const Fx *fx, const Config *c, const Weather *w);
int fx_active_snow(const Fx *fx, const Config *c, const Weather *w);
int fx_active_wind(const Fx *fx, const Config *c, const Weather *w);
int fx_active_fog(const Fx *fx, const Config *c, const Weather *w);
float fx_wind_slant(const Weather *w); // px/s sideways push from wind

// Update particle positions. wind_speed is raw weather wind (km/h or mph).
void fx_update(Fx *fx, const Config *c, const Weather *w, float dt);

// Render everything: tints first, then particles, then lightning flash.
void fx_render(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w);
