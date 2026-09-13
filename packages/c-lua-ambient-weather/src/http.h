// http.h — tiny GET helper over libcurl (shared by geocode + forecast).
#pragma once

#include <stddef.h>

typedef struct {
    char *data; // NUL-terminated response body
    size_t len;
    size_t cap;
} MemBuf;

// GET url into out (timeouts + follow redirects). 0 on HTTP 200, else != 0.
// Frees any partial body on failure.
int http_get(const char *url, MemBuf *out);
void http_free(MemBuf *buf);
