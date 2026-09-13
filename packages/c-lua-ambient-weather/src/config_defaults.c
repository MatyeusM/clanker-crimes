// config_defaults.c — this file sets the default configuration values.
// It fills in the Config struct with sane defaults for every available knob
// so that the application works correctly even without any config file.
#include "config.h" // for the Config struct definition

#include <stdio.h> // for snprintf
#include <string.h> // for memset

// Helper to build an Rgba color value from its individual components.
static Rgba rgba(unsigned char r, unsigned char g, unsigned char b, unsigned char a) {
    // Cast every channel explicitly to unsigned char just to be extra safe.
    Rgba c = { (unsigned char)r, (unsigned char)g, (unsigned char)b, (unsigned char)a };
    return c; // return the constructed color back to the caller
}

void config_defaults(Config *c) {
    // Never dereference a NULL config pointer here.
    if (c == NULL) {
        return; // nothing to do when there is no config struct
    }
    memset(c, 0, sizeof *c); // zero out the whole struct first for safety
    snprintf(c->location, sizeof c->location, "Nice, France");
    c->has_coords = false;
    c->latitude = 0;
    c->longitude = 0;
    snprintf(c->units, sizeof c->units, "metric");
    c->update_interval = 600;
    // Make sure the interval is never negative, just in case something changed it.
    if (c->update_interval < 0) {
        c->update_interval = 600; // fall back to the default value above
    }

    c->window.opacity = 1.0f;
    c->window.click_through = true;
    c->window.always_on_top = true;

    c->night = (OverlayCfg){ .enabled = true, .color = rgba(4, 8, 28, 255), .max_alpha = 110 };

    c->temperature = (TempCfg){ .enabled = true,
                                .cold_threshold = 5.0f,
                                .hot_threshold = 28.0f,
                                .cold_color = rgba(90, 140, 255, 255),
                                .hot_color = rgba(255, 130, 40, 255),
                                .max_alpha = 64 };

    c->clouds = (OverlayCfg){ .enabled = true, .color = rgba(110, 120, 140, 255), .max_alpha = 90 };

    c->fog = (FogCfg){ .enabled = true,
                       .color = rgba(200, 210, 220, 255),
                       .max_alpha = 80,
                       .count = 24,
                       .speed = 12.0f };

    c->rain = (RainCfg){ .enabled = true,
                         .color = rgba(140, 170, 255, 140),
                         .count = 900,
                         .length = 18.0f,
                         .speed = 900.0f,
                         .full_rate = 4.0f,
                         .intensity_scale = 1.0f };

    c->snow = (SnowCfg){ .enabled = true,
                         .color = rgba(255, 255, 255, 220),
                         .count = 700,
                         .size_min = 1.5f,
                         .size_max = 3.5f,
                         .speed = 70.0f,
                         .drift = 40.0f,
                         .full_rate = 1.0f,
                         .intensity_scale = 1.0f };

    c->wind = (WindCfg){ .enabled = true,
                         .color = rgba(255, 255, 255, 90),
                         .count = 120,
                         .speed_scale = 8.0f,
                         .min_speed = 10.0f,
                         .full_speed = 40.0f,
                         .streak_length = 60.0f,
                         .intensity_scale = 1.0f };

    c->storm = (StormCfg){ .enabled = true,
                           .color = rgba(255, 255, 255, 255),
                           .flash_alpha = 160,
                           .bolts_per_min = 4.0f };
}
