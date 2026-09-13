// fx.c — this file holds the particle state and the simulation step.
// Particle positions wrap around the viewport so any window size works.
// The actual drawing happens over in fx_render.c, not in this file.
#include "fx.h"

#include <math.h>
#include <stdlib.h>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

static unsigned long long rng_next(unsigned long long *s) {
    unsigned long long x = *s != 0 ? *s : 0x9E3779B97F4A7C15ULL; // xorshift64*
    x ^= x >> 12;
    x ^= x << 25;
    x ^= x >> 27;
    *s = x;
    return x * 0x2545F4914F6CDD1DULL;
}

static float frand(unsigned long long *s) {
    return (float)((rng_next(s) >> 11) * (1.0 / 9007199254740992.0));
}

static float clamp01f(float x) {
    if (x < 0.0f) {
        return 0.0f;
    }
    if (x > 1.0f) {
        return 1.0f;
    }
    return x;
}

static float wrapf(float v, float m) {
    if (m <= 0.0f) {
        return 0.0f;
    }
    v = fmodf(v, m);
    return v < 0.0f ? v + m : v;
}

void fx_init(Fx *fx, int w, int h) {
    // The FX state pointer must always be valid.
    if (fx == NULL) {
        return; // nothing to initialize without a state struct
    }
    fx->w = w > 0 ? w : 1280;
    fx->h = h > 0 ? h : 720;
    fx->storm_flash = 0.0f;
    fx->storm_timer = 2.0f;
    fx->rng = 0x123456789ABCULL;
    for (int i = 0; i < FX_MAX_RAIN; i++) {
        fx->rain[i] = (RainDrop){ .x = frand(&fx->rng) * (float)fx->w,
                                  .y = frand(&fx->rng) * (float)fx->h,
                                  .speed = 0.7f + frand(&fx->rng) * 0.6f,
                                  .len = 0.8f + frand(&fx->rng) * 0.5f };
    }
    for (int i = 0; i < FX_MAX_SNOW; i++) {
        fx->snow[i] = (SnowFlake){ .x = frand(&fx->rng) * (float)fx->w,
                                   .y = frand(&fx->rng) * (float)fx->h,
                                   .r = 1.0f,
                                   .speed = 0.6f + frand(&fx->rng) * 0.8f,
                                   .phase = frand(&fx->rng) * (float)(2.0 * M_PI),
                                   .sway = 0.5f + frand(&fx->rng) * 1.5f };
    }
    for (int i = 0; i < FX_MAX_WIND; i++) {
        fx->wind[i] = (WindStreak){ .x = frand(&fx->rng) * (float)fx->w,
                                    .y = frand(&fx->rng) * (float)fx->h,
                                    .speed = 0.7f + frand(&fx->rng) * 0.6f,
                                    .len = 0.8f + frand(&fx->rng) * 0.4f };
    }
    for (int i = 0; i < FX_MAX_FOG; i++) {
        fx->fog[i] = (FogPuff){ .x = frand(&fx->rng) * (float)fx->w,
                                 .y = frand(&fx->rng) * (float)fx->h,
                                 .r = 120.0f + frand(&fx->rng) * 220.0f,
                                 .speed = 0.5f + frand(&fx->rng),
                                 .alpha = 0.4f + frand(&fx->rng) * 0.6f };
    }
}

void fx_resize(Fx *fx, int w, int h) {
    if (w > 0) {
        fx->w = w;
    }
    if (h > 0) {
        fx->h = h;
    }
}

// Active particle counts, shared by update + render.
int fx_active_rain(const Fx *fx, const Config *c, const Weather *w) {
    (void)fx;
    int n = c->rain.enabled ? (int)(c->rain.count * clamp01f(weather_rain_intensity(w, c))) : 0;
    return n > FX_MAX_RAIN ? FX_MAX_RAIN : n;
}

int fx_active_snow(const Fx *fx, const Config *c, const Weather *w) {
    (void)fx;
    int n = c->snow.enabled ? (int)(c->snow.count * clamp01f(weather_snow_intensity(w, c))) : 0;
    return n > FX_MAX_SNOW ? FX_MAX_SNOW : n;
}

int fx_active_wind(const Fx *fx, const Config *c, const Weather *w) {
    (void)fx;
    int n = c->wind.enabled ? (int)(c->wind.count * clamp01f(weather_wind_intensity(w, c))) : 0;
    return n > FX_MAX_WIND ? FX_MAX_WIND : n;
}

int fx_active_fog(const Fx *fx, const Config *c, const Weather *w) {
    (void)fx;
    int n = c->fog.enabled ? (int)(c->fog.count * clamp01f(weather_fog_factor(w))) : 0;
    return n > FX_MAX_FOG ? FX_MAX_FOG : n;
}

float fx_wind_slant(const Weather *w) {
    float raw = (w != NULL && w->valid) ? (float)w->wind_speed : 0.0f;
    return raw * 6.0f; // km/h (or mph) -> px/s sideways push
}

void fx_update(Fx *fx, const Config *c, const Weather *w, float dt) {
    // Validate every incoming pointer before use.
    if (fx == NULL || c == NULL) {
        return; // cannot simulate anything without state and config
    }
    if (dt <= 0.0f || dt > 0.25f) {
        dt = (float)0.016; // fall back to a fixed step, cast to be safe
    }
    float W = (float)fx->w, H = (float)fx->h;
    float slant = fx_wind_slant(w);
    float wind_raw = (w != NULL && w->valid) ? (float)w->wind_speed : 0.0f;

    int nrain = fx_active_rain(fx, c, w);
    for (int i = 0; i < nrain; i++) {
        RainDrop *d = &fx->rain[i];
        d->y += c->rain.speed * d->speed * dt;
        d->x += slant * dt * d->speed;
        if (d->y > H + 40.0f) {
            d->y = -40.0f;
            d->x = frand(&fx->rng) * (W + 80.0f) - 40.0f;
        }
        d->x = wrapf(d->x, W + 80.0f) - 40.0f;
    }

    static float global_timer = 0.0f; // sway clock for snow
    global_timer += dt;
    int nsnow = fx_active_snow(fx, c, w);
    for (int i = 0; i < nsnow; i++) {
        SnowFlake *f = &fx->snow[i];
        f->y += c->snow.speed * f->speed * dt;
        f->x += (slant * 0.15f + sinf(global_timer * f->sway + f->phase) * c->snow.drift) * dt;
        if (f->y > H + 10.0f) {
            f->y = -10.0f;
            f->x = frand(&fx->rng) * W;
        }
        f->x = wrapf(f->x, W);
    }

    int nwind = fx_active_wind(fx, c, w);
    for (int i = 0; i < nwind; i++) {
        WindStreak *s = &fx->wind[i];
        s->x += (200.0f + wind_raw * c->wind.speed_scale) * s->speed * dt;
        if (s->x - c->wind.streak_length * s->len > W) {
            s->x = -c->wind.streak_length * s->len;
            s->y = frand(&fx->rng) * H;
        }
    }

    int nfog = fx_active_fog(fx, c, w);
    for (int i = 0; i < nfog; i++) {
        FogPuff *p = &fx->fog[i];
        p->x += (c->fog.speed * p->speed + slant * 0.05f) * dt;
        p->x = wrapf(p->x, W + p->r * 2.0f) - p->r;
    }

    // Lightning: decaying flash + poisson-ish timer during thunderstorms.
    fx->storm_flash *= expf(-6.0f * dt);
    if (fx->storm_flash < 0.01f) {
        fx->storm_flash = 0.0f;
    }
    if (c->storm.enabled && weather_is_thunder(w)) {
        fx->storm_timer -= dt;
        if (fx->storm_timer <= 0.0f) {
            fx->storm_flash = 0.7f + frand(&fx->rng) * 0.3f;
            float bpm = c->storm.bolts_per_min > 0.0f ? c->storm.bolts_per_min : 4.0f;
            fx->storm_timer = (60.0f / bpm) * (0.4f + frand(&fx->rng) * 1.2f);
        }
    } else {
        fx->storm_timer = 1.5f;
    }
}
