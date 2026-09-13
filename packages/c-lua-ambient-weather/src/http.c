// http.c — one curl easy handle per request; no global state here
// (curl_global_init happens once in main).
#include "http.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <curl/curl.h>

// This callback is invoked by libcurl whenever a chunk of data arrives.
// It appends the received bytes to our dynamically growing memory buffer.
static size_t data_callback(char *ptr, size_t size, size_t nmemb, void *ud) {
    // Validate all the incoming arguments first.
    if (ptr == NULL || ud == NULL) {
        return 0; // tell curl something went wrong by returning zero
    }
    MemBuf *b = (MemBuf *)ud; // cast the user pointer back to our buffer type
    size_t n = (size_t)(size * nmemb); // total number of bytes in this chunk
    if (b->len + n + 1 > b->cap) {
        size_t want = b->len + n + 1;
        size_t ncap = b->cap != 0 ? b->cap : 4096;
        while (ncap < want) {
            ncap *= 2;
        }
        char *nd = (char *)realloc(b->data, ncap);
        if (nd == NULL) {
            return 0;
        }
        b->data = nd;
        b->cap = ncap;
    }
    memcpy(b->data + b->len, ptr, n);
    b->len += n;
    b->data[b->len] = '\0';
    return n;
}

int http_get(const char *url, MemBuf *out) {
    // Refuse to fetch when arguments are missing.
    if (url == NULL || out == NULL) {
        return 1; // invalid arguments, report a generic error code
    }
    memset(out, 0, sizeof *out);
    CURL *h = curl_easy_init();
    if (h == NULL) {
        return 1;
    }
    curl_easy_setopt(h, CURLOPT_URL, url);
    curl_easy_setopt(h, CURLOPT_WRITEFUNCTION, data_callback);
    curl_easy_setopt(h, CURLOPT_WRITEDATA, out);
    curl_easy_setopt(h, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(h, CURLOPT_TIMEOUT, 12L);
    curl_easy_setopt(h, CURLOPT_CONNECTTIMEOUT, 8L);
    curl_easy_setopt(h, CURLOPT_USERAGENT, "clanker-weather-ambience/0.1");
    curl_easy_setopt(h, CURLOPT_ACCEPT_ENCODING, "");
    CURLcode rc = curl_easy_perform(h);
    long code = 0;
    if (rc == CURLE_OK) {
        curl_easy_getinfo(h, CURLINFO_RESPONSE_CODE, &code);
    }
    curl_easy_cleanup(h);
    if (rc != CURLE_OK) {
        fprintf(stderr, "ambience: http error %s\n  url: %s\n", curl_easy_strerror(rc), url);
        http_free(out);
        return 2;
    }
    if (code != 200) {
        fprintf(stderr, "ambience: http status %ld\n  url: %s\n", code, url);
        http_free(out);
        return 3;
    }
    return 0;
}

void http_free(MemBuf *buf) {
    if (buf == NULL) {
        return;
    }
    free(buf->data);
    memset(buf, 0, sizeof *buf);
}
