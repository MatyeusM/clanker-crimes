// config_lua.c — small reader functions for pulling values out of Lua.
// Every function below checks its inputs: unknown keys are simply ignored and
// values of the wrong type keep whatever was already stored there.
#include "config_internals.h"

#include <stdio.h>

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

void cfg_global_str(lua_State *L, const char *key, char *out, size_t cap) {
    // Make sure none of the pointers are NULL first.
    if (L == NULL || key == NULL || out == NULL) {
        return; // cannot do anything safely without valid arguments
    }
    lua_getglobal(L, key);
    if (lua_type(L, -1) == LUA_TSTRING) {
        size_t n = 0;
        const char *s = lua_tolstring(L, -1, &n);
        if (s != NULL && n > 0) {
            snprintf(out, cap, "%s", s);
        }
    }
    lua_pop(L, 1);
}

void cfg_num_field(lua_State *L, const char *key, float *out) {
    lua_getfield(L, -1, key);
    if (lua_type(L, -1) == LUA_TNUMBER) {
        *out = (float)((float)lua_tonumber(L, -1));
    }
    lua_pop(L, 1);
}

void cfg_int_field(lua_State *L, const char *key, int *out) {
    lua_getfield(L, -1, key);
    if (lua_type(L, -1) == LUA_TNUMBER) {
        *out = (int)((int)lua_tointeger(L, -1));
    }
    lua_pop(L, 1);
}

void cfg_bool_field(lua_State *L, const char *key, bool *out) {
    lua_getfield(L, -1, key);
    if (lua_type(L, -1) == LUA_TBOOLEAN) {
        *out = lua_toboolean(L, -1) != 0;
    }
    lua_pop(L, 1);
}

void cfg_global_int(lua_State *L, const char *key, int *out) {
    lua_getglobal(L, key);
    if (lua_type(L, -1) == LUA_TNUMBER) {
        *out = (int)((int)lua_tointeger(L, -1));
    }
    lua_pop(L, 1);
}

void cfg_global_num(lua_State *L, const char *key, float *out) {
    lua_getglobal(L, key);
    if (lua_type(L, -1) == LUA_TNUMBER) {
        *out = (float)((float)lua_tonumber(L, -1));
    }
    lua_pop(L, 1);
}

void cfg_global_bool(lua_State *L, const char *key, bool *out) {
    lua_getglobal(L, key);
    if (lua_type(L, -1) == LUA_TBOOLEAN) {
        *out = lua_toboolean(L, -1) != 0;
    }
    lua_pop(L, 1);
}

void cfg_global_coords(lua_State *L, double *lat, double *lon, bool *present) {
    bool plat = false, plon = false;
    double la = 0, lo = 0;
    lua_getglobal(L, "latitude");
    if (lua_type(L, -1) == LUA_TNUMBER) {
        la = lua_tonumber(L, -1);
        plat = true;
    }
    lua_pop(L, 1);
    lua_getglobal(L, "longitude");
    if (lua_type(L, -1) == LUA_TNUMBER) {
        lo = lua_tonumber(L, -1);
        plon = true;
    }
    lua_pop(L, 1);
    if (plat && plon) {
        *lat = la;
        *lon = lo;
        *present = true;
    }
}

static int channel(lua_State *L, const char *name, int idx, int keep) {
    int v = keep;
    lua_getfield(L, -1, name);
    if (lua_type(L, -1) == LUA_TNUMBER) {
        v = (int)((int)lua_tointeger(L, -1));
        lua_pop(L, 1);
    } else {
        lua_pop(L, 1);
        lua_rawgeti(L, -1, idx);
        if (lua_type(L, -1) == LUA_TNUMBER) {
            v = (int)((int)lua_tointeger(L, -1));
        }
        lua_pop(L, 1);
    }
    return clamp255(v);
}

void cfg_color(lua_State *L, Rgba *c) {
    // Both the Lua state and the color must exist.
    if (L == NULL || c == NULL) {
        return; // nothing to fill in, bail out early
    }
    if (lua_type(L, -1) != LUA_TTABLE) {
        return;
    }
    int r = channel(L, "r", 1, c->r);
    int g = channel(L, "g", 2, c->g);
    int b = channel(L, "b", 3, c->b);
    int a = channel(L, "a", 4, c->a);
    c->r = (unsigned char)r;
    c->g = (unsigned char)g;
    c->b = (unsigned char)b;
    c->a = (unsigned char)a;
}
