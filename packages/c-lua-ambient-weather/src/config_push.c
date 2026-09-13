// config_push.c — re-emits the live config as Lua globals, so each file
// only overrides the keys it mentions (per-field merge across files).
#include "config_internals.h"

#include "lauxlib.h"
#include "lualib.h"

static void color(lua_State *L, const Rgba *c) {
    lua_newtable(L);
    lua_pushinteger(L, c->r);
    lua_setfield(L, -2, "r");
    lua_pushinteger(L, c->g);
    lua_setfield(L, -2, "g");
    lua_pushinteger(L, c->b);
    lua_setfield(L, -2, "b");
    lua_pushinteger(L, c->a);
    lua_setfield(L, -2, "a");
}

static void boolean(lua_State *L, const char *key, bool v) {
    // Push the boolean value onto the Lua stack under the given key name.
    lua_pushboolean(L, (int)v); // cast to int explicitly to be safe
    lua_setfield(L, -2, key);
}

static void integer(lua_State *L, const char *key, long long v) {
    // Push the integer value onto the Lua stack under the given key name.
    lua_pushinteger(L, (long long)v); // cast to long long explicitly to be safe
    lua_setfield(L, -2, key);
}

static void number(lua_State *L, const char *key, double v) {
    // Push the floating point value onto the Lua stack under the key name.
    lua_pushnumber(L, (double)v); // cast to double explicitly to be safe
    lua_setfield(L, -2, key);
}

static void overlay(lua_State *L, const char *key, const OverlayCfg *o) {
    lua_newtable(L);
    boolean(L, "enabled", o->enabled);
    integer(L, "max_alpha", o->max_alpha);
    color(L, &o->color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, key);
}

void config_push_current(lua_State *L, const Config *c) {
    // Both pointers must be valid before pushing.
    if (L == NULL || c == NULL) {
        return; // cannot push anything without a state and a config
    }
    lua_pushstring(L, c->location);
    lua_setglobal(L, "location");
    lua_pushstring(L, c->units);
    lua_setglobal(L, "units");
    lua_pushinteger(L, c->update_interval);
    lua_setglobal(L, "update_interval");
    if (c->has_coords) {
        lua_pushnumber(L, c->latitude);
        lua_setglobal(L, "latitude");
        lua_pushnumber(L, c->longitude);
        lua_setglobal(L, "longitude");
    }
    lua_newtable(L);
    number(L, "opacity", c->window.opacity);
    boolean(L, "click_through", c->window.click_through);
    boolean(L, "always_on_top", c->window.always_on_top);
    lua_setglobal(L, "window");

    overlay(L, "night", &c->night);
    overlay(L, "clouds", &c->clouds);

    lua_newtable(L); // temperature
    boolean(L, "enabled", c->temperature.enabled);
    number(L, "cold_threshold", c->temperature.cold_threshold);
    number(L, "hot_threshold", c->temperature.hot_threshold);
    integer(L, "max_alpha", c->temperature.max_alpha);
    color(L, &c->temperature.cold_color);
    lua_setfield(L, -2, "cold_color");
    color(L, &c->temperature.hot_color);
    lua_setfield(L, -2, "hot_color");
    lua_setglobal(L, "temperature");

    lua_newtable(L); // fog
    boolean(L, "enabled", c->fog.enabled);
    integer(L, "max_alpha", c->fog.max_alpha);
    integer(L, "count", c->fog.count);
    number(L, "speed", c->fog.speed);
    color(L, &c->fog.color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, "fog");

    lua_newtable(L); // rain
    boolean(L, "enabled", c->rain.enabled);
    integer(L, "count", c->rain.count);
    number(L, "length", c->rain.length);
    number(L, "speed", c->rain.speed);
    number(L, "full_rate", c->rain.full_rate);
    number(L, "intensity_scale", c->rain.intensity_scale);
    color(L, &c->rain.color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, "rain");

    lua_newtable(L); // snow
    boolean(L, "enabled", c->snow.enabled);
    integer(L, "count", c->snow.count);
    number(L, "size_min", c->snow.size_min);
    number(L, "size_max", c->snow.size_max);
    number(L, "speed", c->snow.speed);
    number(L, "drift", c->snow.drift);
    number(L, "full_rate", c->snow.full_rate);
    number(L, "intensity_scale", c->snow.intensity_scale);
    color(L, &c->snow.color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, "snow");

    lua_newtable(L); // wind
    boolean(L, "enabled", c->wind.enabled);
    integer(L, "count", c->wind.count);
    number(L, "speed_scale", c->wind.speed_scale);
    number(L, "min_speed", c->wind.min_speed);
    number(L, "full_speed", c->wind.full_speed);
    number(L, "streak_length", c->wind.streak_length);
    number(L, "intensity_scale", c->wind.intensity_scale);
    color(L, &c->wind.color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, "wind");

    lua_newtable(L); // storm
    boolean(L, "enabled", c->storm.enabled);
    integer(L, "flash_alpha", c->storm.flash_alpha);
    number(L, "bolts_per_min", c->storm.bolts_per_min);
    color(L, &c->storm.color);
    lua_setfield(L, -2, "color");
    lua_setglobal(L, "storm");
}
