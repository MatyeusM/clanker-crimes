// config_apply.c — this file merges the Lua globals into the live config.
// It is called once per config file, so values accumulate across files.
#include "config_internals.h"

#include "lauxlib.h"

static int clamp255(int x) {
    if (x < 0) {
        return 0;
    }
    if (x > 255) {
        return 255;
    }
    return x;
}

static void overlay(lua_State *L, const char *key, OverlayCfg *o) {
    lua_getglobal(L, key);
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &o->enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &o->color);
        lua_pop(L, 1);
        int temp_value = o->max_alpha;
        cfg_int_field(L, "max_alpha", &temp_value);
        o->max_alpha = (unsigned char)clamp255(temp_value);
    }
    lua_pop(L, 1);
}

static void apply_window(lua_State *L, Config *c) {
    lua_getglobal(L, "window");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_num_field(L, "opacity", &c->window.opacity);
        if (c->window.opacity < 0.0f) {
            c->window.opacity = 0.0f;
        }
        if (c->window.opacity > 1.0f) {
            c->window.opacity = 1.0f;
        }
        cfg_bool_field(L, "click_through", &c->window.click_through);
        cfg_bool_field(L, "always_on_top", &c->window.always_on_top);
    }
    lua_pop(L, 1);
}

static void apply_temperature(lua_State *L, Config *c) {
    lua_getglobal(L, "temperature");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->temperature.enabled);
        cfg_num_field(L, "cold_threshold", &c->temperature.cold_threshold);
        cfg_num_field(L, "hot_threshold", &c->temperature.hot_threshold);
        lua_getfield(L, -1, "cold_color");
        cfg_color(L, &c->temperature.cold_color);
        lua_pop(L, 1);
        lua_getfield(L, -1, "hot_color");
        cfg_color(L, &c->temperature.hot_color);
        lua_pop(L, 1);
        int temp_value = c->temperature.max_alpha;
        cfg_int_field(L, "max_alpha", &temp_value);
        c->temperature.max_alpha = (unsigned char)clamp255(temp_value);
    }
    lua_pop(L, 1);
}

static void apply_fog(lua_State *L, Config *c) {
    lua_getglobal(L, "fog");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->fog.enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &c->fog.color);
        lua_pop(L, 1);
        int temp_value = c->fog.max_alpha;
        cfg_int_field(L, "max_alpha", &temp_value);
        c->fog.max_alpha = (unsigned char)clamp255(temp_value);
        cfg_int_field(L, "count", &c->fog.count);
        cfg_num_field(L, "speed", &c->fog.speed);
        if (c->fog.count < 0) {
            c->fog.count = 0;
        }
        if (c->fog.count > 256) {
            c->fog.count = 256;
        }
    }
    lua_pop(L, 1);
}

static void apply_rain(lua_State *L, Config *c) {
    lua_getglobal(L, "rain");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->rain.enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &c->rain.color);
        lua_pop(L, 1);
        cfg_int_field(L, "count", &c->rain.count);
        cfg_num_field(L, "length", &c->rain.length);
        cfg_num_field(L, "speed", &c->rain.speed);
        cfg_num_field(L, "full_rate", &c->rain.full_rate);
        cfg_num_field(L, "intensity_scale", &c->rain.intensity_scale);
        if (c->rain.count < 0) {
            c->rain.count = 0;
        }
        if (c->rain.count > 4000) {
            c->rain.count = 4000;
        }
        if (c->rain.full_rate <= 0.0f) {
            c->rain.full_rate = 4.0f;
        }
    }
    lua_pop(L, 1);
}

static void apply_snow(lua_State *L, Config *c) {
    lua_getglobal(L, "snow");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->snow.enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &c->snow.color);
        lua_pop(L, 1);
        cfg_int_field(L, "count", &c->snow.count);
        cfg_num_field(L, "size_min", &c->snow.size_min);
        cfg_num_field(L, "size_max", &c->snow.size_max);
        cfg_num_field(L, "speed", &c->snow.speed);
        cfg_num_field(L, "drift", &c->snow.drift);
        cfg_num_field(L, "full_rate", &c->snow.full_rate);
        cfg_num_field(L, "intensity_scale", &c->snow.intensity_scale);
        if (c->snow.count < 0) {
            c->snow.count = 0;
        }
        if (c->snow.count > 4000) {
            c->snow.count = 4000;
        }
        if (c->snow.full_rate <= 0.0f) {
            c->snow.full_rate = 1.0f;
        }
    }
    lua_pop(L, 1);
}

static void apply_wind(lua_State *L, Config *c) {
    lua_getglobal(L, "wind");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->wind.enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &c->wind.color);
        lua_pop(L, 1);
        cfg_int_field(L, "count", &c->wind.count);
        cfg_num_field(L, "speed_scale", &c->wind.speed_scale);
        cfg_num_field(L, "min_speed", &c->wind.min_speed);
        cfg_num_field(L, "full_speed", &c->wind.full_speed);
        cfg_num_field(L, "streak_length", &c->wind.streak_length);
        cfg_num_field(L, "intensity_scale", &c->wind.intensity_scale);
        if (c->wind.count < 0) {
            c->wind.count = 0;
        }
        if (c->wind.count > 1000) {
            c->wind.count = 1000;
        }
    }
    lua_pop(L, 1);
}

static void apply_storm(lua_State *L, Config *c) {
    lua_getglobal(L, "storm");
    if (lua_type(L, -1) == LUA_TTABLE) {
        cfg_bool_field(L, "enabled", &c->storm.enabled);
        lua_getfield(L, -1, "color");
        cfg_color(L, &c->storm.color);
        lua_pop(L, 1);
        int temp_value = c->storm.flash_alpha;
        cfg_int_field(L, "flash_alpha", &temp_value);
        c->storm.flash_alpha = (unsigned char)clamp255(temp_value);
        cfg_num_field(L, "bolts_per_min", &c->storm.bolts_per_min);
    }
    lua_pop(L, 1);
}

void config_apply(lua_State *L, Config *c) {
    // Refuse to work with NULL arguments.
    if (L == NULL || c == NULL) {
        return; // cannot apply anything without a state and a config
    }
    cfg_global_str(L, "location", c->location, sizeof c->location);
    cfg_global_str(L, "units", c->units, sizeof c->units);
    cfg_global_int(L, "update_interval", &c->update_interval);
    if (c->update_interval < 30) {
        c->update_interval = 30;
    }
    double lat = 0, lon = 0;
    bool has = false;
    cfg_global_coords(L, &lat, &lon, &has);
    if (has) {
        c->latitude = lat;
        c->longitude = lon;
        c->has_coords = true;
    }
    apply_window(L, c);
    overlay(L, "night", &c->night);
    apply_temperature(L, c);
    overlay(L, "clouds", &c->clouds);
    apply_fog(L, c);
    apply_rain(L, c);
    apply_snow(L, c);
    apply_wind(L, c);
    apply_storm(L, c);
    // Legacy flat aliases for the common knobs.
    cfg_global_bool(L, "night_enabled", &c->night.enabled);
    cfg_global_bool(L, "rain_enabled", &c->rain.enabled);
    cfg_global_bool(L, "snow_enabled", &c->snow.enabled);
    cfg_global_bool(L, "wind_enabled", &c->wind.enabled);
    cfg_global_num(L, "rain_intensity", &c->rain.intensity_scale);
    cfg_global_num(L, "snow_intensity", &c->snow.intensity_scale);
    cfg_global_num(L, "wind_intensity", &c->wind.intensity_scale);
}
