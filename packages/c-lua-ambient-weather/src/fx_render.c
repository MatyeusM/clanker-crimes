// fx_render.c — this file draws everything onto the SDL renderer.
// The order matters: fullscreen tints first, then particles on top,
// and the lightning flash is drawn last so it covers the whole scene.
#include "fx.h"

#include <math.h>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

static float clamp01f(float x) {
    if (x < 0.0f) {
        return 0.0f;
    }
    if (x > 1.0f) {
        return 1.0f;
    }
    return x;
}

static void fill_screen(SDL_Renderer *ren, Rgba col, unsigned char alpha) {
    if (alpha == 0) {
        return;
    }
    SDL_SetRenderDrawColor(ren, col.r, col.g, col.b, alpha);
    SDL_RenderFillRect(ren, NULL);
}

static void render_temperature(SDL_Renderer *ren, const Config *c, const Weather *w) {
    if (!c->temperature.enabled || w == NULL || !w->valid) {
        return;
    }
    float t = (float)w->temperature;
    if (t < c->temperature.cold_threshold) {
        float k = clamp01f((c->temperature.cold_threshold - t) / 15.0f);
        fill_screen(ren, c->temperature.cold_color,
                    (unsigned char)(c->temperature.max_alpha * (0.3f + 0.7f * k)));
    } else if (t > c->temperature.hot_threshold) {
        float k = clamp01f((t - c->temperature.hot_threshold) / 12.0f);
        fill_screen(ren, c->temperature.hot_color,
                    (unsigned char)(c->temperature.max_alpha * (0.3f + 0.7f * k)));
    }
}

static void render_fog(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w) {
    if (!c->fog.enabled) {
        return;
    }
    float fog_i = weather_fog_factor(w);
    int n = fx_active_fog(fx, c, w);
    // Loop over every active fog puff and draw it onto the screen.
    for (int i = 0; i < n; i++) {
        FogPuff *p = &fx->fog[i]; // get a pointer to the current puff
        unsigned char base =
            (unsigned char)(c->fog.max_alpha * p->alpha * (0.25f + 0.75f * fog_i));
        for (int ring = 3; ring >= 1; ring--) { // 3 rings fake a soft radial blob
            float rr = p->r * (float)ring / 3.0f; // cast ring to float to be safe
            unsigned char a = (unsigned char)(base / (4 - ring));
            if (a == 0) {
                continue;
            }
            SDL_SetRenderDrawColor(ren, c->fog.color.r, c->fog.color.g, c->fog.color.b, a);
            const int segs = 48;
            for (int s = 0; s < segs; s++) {
                float a0 = (float)s / segs * (float)(2.0 * M_PI);
                float a1 = (float)(s + 1) / segs * (float)(2.0 * M_PI);
                SDL_RenderLine(ren, p->x + cosf(a0) * rr, p->y + sinf(a0) * rr * 0.55f,
                               p->x + cosf(a1) * rr, p->y + sinf(a1) * rr * 0.55f);
            }
        }
    }
}

static void render_rain(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w) {
    if (!c->rain.enabled) {
        return;
    }
    int n = fx_active_rain(fx, c, w);
    if (n == 0) {
        return;
    }
    float dx = fx_wind_slant(w) * 0.02f * c->rain.length; // wind slant
    SDL_SetRenderDrawColor(ren, c->rain.color.r, c->rain.color.g, c->rain.color.b,
                           c->rain.color.a);
    for (int i = 0; i < n; i++) {
        RainDrop *d = &fx->rain[i];
        SDL_RenderLine(ren, d->x, d->y, d->x - dx, d->y - c->rain.length * d->len);
    }
}

static void render_snow(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w) {
    if (!c->snow.enabled) {
        return;
    }
    int n = fx_active_snow(fx, c, w);
    if (n == 0) {
        return;
    }
    float span = c->snow.size_max - c->snow.size_min;
    if (span < 0.1f) {
        span = 0.1f;
    }
    SDL_SetRenderDrawColor(ren, c->snow.color.r, c->snow.color.g, c->snow.color.b,
                           c->snow.color.a);
    for (int i = 0; i < n; i++) {
        SnowFlake *f = &fx->snow[i]; // get a pointer to the current flake
        // Compute a deterministic pseudo-size from the index so flakes do not shimmer.
        float r = c->snow.size_min + span * (float)(((i * 2654435761u) % 1000) / 1000.0f);
        SDL_RenderPoint(ren, f->x, f->y); // draw the center point of the flake
        if (r >= 2.0f) { // cross reads as a flake at small sizes
            SDL_RenderLine(ren, f->x - r, f->y, f->x + r, f->y);
            SDL_RenderLine(ren, f->x, f->y - r, f->x, f->y + r);
        }
    }
}

static void render_wind(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w) {
    if (!c->wind.enabled) {
        return;
    }
    int n = fx_active_wind(fx, c, w);
    if (n == 0) {
        return;
    }
    SDL_SetRenderDrawColor(ren, c->wind.color.r, c->wind.color.g, c->wind.color.b,
                           c->wind.color.a);
    for (int i = 0; i < n; i++) {
        WindStreak *s = &fx->wind[i];
        SDL_RenderLine(ren, s->x, s->y, s->x + c->wind.streak_length * s->len, s->y);
    }
}

void fx_render(SDL_Renderer *ren, Fx *fx, const Config *c, const Weather *w) {
    // Every pointer must be valid before drawing.
    if (ren == NULL || fx == NULL || c == NULL) {
        return; // cannot render anything without renderer, state and config
    }
    // Note: w may be invalid (offline), the helpers below handle that case.
    bool valid = w != NULL && w->valid;
    if (c->night.enabled && valid && !w->is_day) {
        fill_screen(ren, c->night.color, c->night.max_alpha);
    }
    render_temperature(ren, c, w);
    if (c->clouds.enabled && valid) {
        fill_screen(ren, c->clouds.color,
                    (unsigned char)(c->clouds.max_alpha * weather_cloud_factor(w)));
    }
    render_fog(ren, fx, c, w);
    render_rain(ren, fx, c, w);
    render_snow(ren, fx, c, w);
    render_wind(ren, fx, c, w);
    if (c->storm.enabled && fx->storm_flash > 0.0f) {
        fill_screen(ren, c->storm.color,
                    (unsigned char)(c->storm.flash_alpha * clamp01f(fx->storm_flash)));
    }
}
