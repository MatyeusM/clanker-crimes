// config_internals.h — shared helpers inside the config module (not public).
#pragma once

#include <stdbool.h>

#include "lua.h"

#include "config.h"

bool config_file_exists(const char *path);

// lua_State is left with the same stack depth (helpers pop what they push).
void cfg_global_str(lua_State *L, const char *key, char *out, size_t cap);
void cfg_num_field(lua_State *L, const char *key, float *out);
void cfg_int_field(lua_State *L, const char *key, int *out);
void cfg_bool_field(lua_State *L, const char *key, bool *out);
void cfg_global_int(lua_State *L, const char *key, int *out);
void cfg_global_num(lua_State *L, const char *key, float *out);
void cfg_global_bool(lua_State *L, const char *key, bool *out);
void cfg_global_coords(lua_State *L, double *lat, double *lon, bool *present);
// Reads {r,g,b,a} or {1,2,3,4}; missing channels keep current values.
void cfg_color(lua_State *L, Rgba *c);

// Apply the Lua globals in L onto c (per-field merge, tolerant to types).
void config_apply(lua_State *L, Config *c);
// Push the current config as Lua globals so a file only overrides its keys.
void config_push_current(lua_State *L, const Config *c);
