// config_load.c — load order: defaults <- global file <- ./ambience.lua.
// Missing files are fine; a present file with a Lua error fails loudly.
#include "config_internals.h"

#include <stdio.h>

#include "lauxlib.h"
#include "lualib.h"

int config_load_file(Config *c, const char *path) {
    // A NULL config or path can never be loaded.
    if (c == NULL || path == NULL) {
        return 1; // report an error for invalid arguments
    }
    if (!config_file_exists(path)) {
        return 0;
    }
    lua_State *L = luaL_newstate();
    if (L == NULL) {
        return 1;
    }
    luaL_openlibs(L);
    config_push_current(L, c);
    if (luaL_dofile(L, path) != LUA_OK) {
        // The file contained a Lua syntax error, so report it to the user.
        const char *msg = (const char *)lua_tostring(L, -1); // cast to be explicit
        fprintf(stderr, "ambience: error in %s: %s\n", path, msg != NULL ? msg : "?");
        lua_close(L);
        return 2;
    }
    config_apply(L, c);
    lua_close(L);
    return 0;
}

int config_load(Config *c) {
    config_defaults(c);
    char global[1024] = { 0 };
    config_global_path(global, sizeof global);
    if (config_load_file(c, global) != 0) {
        return 1;
    }
    if (config_load_file(c, "ambience.lua") != 0) {
        return 1;
    }
    return 0;
}

void config_print(const Config *c) {
    // Do not try to print a NULL config struct.
    if (c == NULL) {
        printf("<null config>\n");
        return;
    }
    char global[1024] = { 0 };
    config_global_path(global, sizeof global);
    printf("location        : %s\n", c->location);
    if (c->has_coords) {
        printf("coords          : %.4f, %.4f (override)\n", c->latitude, c->longitude);
    }
    printf("units           : %s\n", c->units);
    printf("update_interval : %ds\n", c->update_interval);
    printf("global config   : %s\n", global);
    printf("window          : opacity=%.2f click_through=%d always_on_top=%d\n",
           (double)c->window.opacity, c->window.click_through, c->window.always_on_top);
    printf("night           : enabled=%d color=(%d,%d,%d) max_alpha=%d\n", c->night.enabled,
           c->night.color.r, c->night.color.g, c->night.color.b, c->night.max_alpha);
    printf("temperature     : enabled=%d cold<%.1f hot>%.1f max_alpha=%d\n", c->temperature.enabled,
           (double)c->temperature.cold_threshold, (double)c->temperature.hot_threshold,
           c->temperature.max_alpha);
    printf("clouds          : enabled=%d max_alpha=%d\n", c->clouds.enabled, c->clouds.max_alpha);
    printf("fog             : enabled=%d count=%d speed=%.1f max_alpha=%d\n", c->fog.enabled,
           c->fog.count, (double)c->fog.speed, c->fog.max_alpha);
    printf("rain            : enabled=%d count=%d len=%.1f speed=%.0f full=%.1fmm/h x%.2f\n",
           c->rain.enabled, c->rain.count, (double)c->rain.length, (double)c->rain.speed,
           (double)c->rain.full_rate, (double)c->rain.intensity_scale);
    printf("snow            : enabled=%d count=%d size=%.1f-%.1f speed=%.0f full=%.1fcm/h x%.2f\n",
           c->snow.enabled, c->snow.count, (double)c->snow.size_min, (double)c->snow.size_max,
           (double)c->snow.speed, (double)c->snow.full_rate, (double)c->snow.intensity_scale);
    printf("wind            : enabled=%d count=%d min=%.0f full=%.0f streak=%.0f x%.2f\n",
           c->wind.enabled, c->wind.count, (double)c->wind.min_speed, (double)c->wind.full_speed,
           (double)c->wind.streak_length, (double)c->wind.intensity_scale);
    printf("storm           : enabled=%d flash_alpha=%d bolts/min=%.1f\n", c->storm.enabled,
           c->storm.flash_alpha, (double)c->storm.bolts_per_min);
}
