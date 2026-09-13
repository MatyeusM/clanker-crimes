// weather.c — geocoding + current forecast via Open-Meteo (free, keyless).
#include "weather.h"

#include <stdio.h>
#include <string.h>
#include <time.h>

#include <curl/curl.h>
#include <yyjson.h>

#include "http.h"

// Helper that reads a number field out of a JSON object, or returns dflt.
// A NULL object or a non-numeric value yields the default.
static double get_num(yyjson_val *obj, const char *key, double dflt) {
    if (obj == NULL || key == NULL) {
        return dflt; // no object to read from, so use the fallback value
    }
    yyjson_val *v = yyjson_obj_get(obj, key);
    return v != NULL && yyjson_is_num(v) ? yyjson_get_num(v) : dflt;
}

static long long get_int(yyjson_val *obj, const char *key, long long dflt) {
    if (obj == NULL) {
        return dflt;
    }
    yyjson_val *v = yyjson_obj_get(obj, key);
    return v != NULL && yyjson_is_num(v) ? yyjson_get_sint(v) : dflt;
}

// Resolve "Nice, France" -> lat/lon via the Open-Meteo geocoding API.
static int do_lookup(const char *place, double *lat, double *lon, char *label, size_t cap) {
    CURL *h = curl_easy_init();
    if (h == NULL) {
        return 1;
    }
    char *esc = curl_easy_escape(h, place, 0);
    if (esc == NULL) {
        curl_easy_cleanup(h);
        return 1;
    }
    char url[1024];
    snprintf(url, sizeof url,
             "https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
             esc);
    curl_free(esc);
    curl_easy_cleanup(h);

    MemBuf b;
    if (http_get(url, &b) != 0) {
        return 2;
    }
    yyjson_doc *doc = yyjson_read(b.data, b.len, 0);
    http_free(&b);
    if (doc == NULL) {
        fprintf(stderr, "ambience: geocode response was not JSON\n");
        return 3;
    }
    yyjson_val *results = yyjson_obj_get(yyjson_doc_get_root(doc), "results");
    yyjson_val *item =
        (results != NULL && yyjson_is_arr(results)) ? yyjson_arr_get_first(results) : NULL;
    if (item == NULL) {
        fprintf(stderr, "ambience: location not found: %s\n", place);
        yyjson_doc_free(doc);
        return 4;
    }
    *lat = get_num(item, "latitude", 0.0);
    *lon = get_num(item, "longitude", 0.0);
    yyjson_val *nm = yyjson_obj_get(item, "name");
    yyjson_val *cc = yyjson_obj_get(item, "country");
    if (yyjson_is_str(nm) && yyjson_is_str(cc)) {
        // Both name and country are strings, so combine them into the label.
        snprintf(label, (size_t)cap, "%s, %s", yyjson_get_str(nm), yyjson_get_str(cc));
    } else if (yyjson_is_str(nm)) {
        snprintf(label, (size_t)cap, "%s", yyjson_get_str(nm));
    } else {
        snprintf(label, (size_t)cap, "%s", place);
    }
    yyjson_doc_free(doc);
    return 0;
}

static void fill_current(yyjson_val *cur, Weather *out) {
    // Both the JSON node and the output must exist.
    if (cur == NULL || out == NULL) {
        return; // nothing to fill in, bail out early to stay safe
    }
    out->temperature = get_num(cur, "temperature_2m", 0.0);
    out->apparent = get_num(cur, "apparent_temperature", out->temperature);
    out->humidity = get_num(cur, "relative_humidity_2m", 50.0);
    out->cloud_cover = get_num(cur, "cloud_cover", 0.0);
    out->wind_speed = get_num(cur, "wind_speed_10m", 0.0);
    out->wind_dir = get_num(cur, "wind_direction_10m", 0.0);
    out->precipitation = get_num(cur, "precipitation", 0.0);
    out->rain = get_num(cur, "rain", 0.0);
    out->showers = get_num(cur, "showers", 0.0);
    out->snowfall = get_num(cur, "snowfall", 0.0);
    out->visibility = get_num(cur, "visibility", 24000.0);
    out->weather_code = (int)get_int(cur, "weather_code", 0);
    out->is_day = (int)get_int(cur, "is_day", 1);
}

int weather_fetch(const Config *cfg, Weather *out) {
    // Refuse to work with NULL arguments here.
    if (cfg == NULL || out == NULL) {
        return 1; // invalid arguments, report a generic error code
    }
    memset(out, 0, sizeof *out);

    double lat = cfg->latitude, lon = cfg->longitude;
    char label[128] = { 0 };
    snprintf(label, sizeof label, "%s", cfg->location);
    if (!cfg->has_coords && do_lookup(cfg->location, &lat, &lon, label, sizeof label) != 0) {
        return 1;
    }

    bool imperial = cfg->units[0] == 'i' || cfg->units[0] == 'I';
    char url[1024];
    snprintf(url, sizeof url,
             "https://api.open-meteo.com/v1/forecast?latitude=%.5f&longitude=%.5f"
             "&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,"
             "precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,"
             "snowfall,rain,showers,visibility"
             "&temperature_unit=%s&wind_speed_unit=%s&timezone=auto",
             lat, lon, imperial ? "fahrenheit" : "celsius", imperial ? "mph" : "kmh");

    MemBuf b;
    if (http_get(url, &b) != 0) {
        return 2;
    }
    yyjson_doc *doc = yyjson_read(b.data, b.len, 0);
    http_free(&b);
    if (doc == NULL) {
        fprintf(stderr, "ambience: forecast response was not JSON\n");
        return 3;
    }
    yyjson_val *cur = yyjson_obj_get(yyjson_doc_get_root(doc), "current");
    if (cur == NULL) {
        fprintf(stderr, "ambience: forecast response has no `current`\n");
        yyjson_doc_free(doc);
        return 4;
    }
    fill_current(cur, out);
    out->latitude = lat;
    out->longitude = lon;
    snprintf(out->place, sizeof out->place, "%s", label);
    out->fetched_at = (long long)time(NULL);
    out->valid = true;
    yyjson_doc_free(doc);
    return 0;
}
