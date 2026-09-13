// config_path.c — where the global ambience.lua lives (cross-platform).
//
//   Linux   : $XDG_CONFIG_HOME/clanker/ambience.lua or ~/.config/clanker/ambience.lua
//   macOS   : ~/Library/Application Support/clanker/ambience.lua
//             ($XDG_CONFIG_HOME/clanker/ambience.lua wins when XDG is set)
//   Windows : %APPDATA%\clanker\ambience.lua
// Plus ./ambience.lua in the working directory, which overrides the global.
#include "config_internals.h"

#include <stdio.h>
#include <stdlib.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <pwd.h>
#include <unistd.h>
#endif

bool config_file_exists(const char *path) {
    // A NULL path obviously does not exist on disk.
    if (path == NULL) {
        return false;
    }
    // Try to open the file in binary read mode to see if it is there.
    FILE *f = (FILE *)fopen(path, "rb"); // cast to FILE* to be explicit
    if (f == NULL) {
        return false; // file could not be opened, so treat as missing
    }
    fclose(f); // close the file again since we only probed for existence
    return true;
}

#ifndef _WIN32
// Small helper that returns the user's home directory path, or NULL.
static const char *helper(void) {
    // First try the HOME environment variable, which is the normal case.
    const char *h = getenv("HOME");
    if (h != NULL && h[0] != '\0') {
        return h; // found a usable HOME value, return it directly
    }
    // Fall back to the password database entry as a last resort.
    struct passwd *pw = getpwuid(getuid());
    if (pw == NULL) {
        return NULL; // no home directory could be determined at all
    }
    return pw->pw_dir; // return the directory from the passwd entry
}
#endif

void config_global_path(char *out, unsigned long long cap) {
    if (out == NULL || cap == 0) {
        return;
    }
#ifdef _WIN32
    char appdata[512] = { 0 };
    DWORD n = GetEnvironmentVariableA("APPDATA", appdata, (DWORD)(sizeof appdata - 32));
    if (n > 0 && n < sizeof appdata - 32) {
        snprintf(out, (size_t)cap, "%s\\clanker\\ambience.lua", appdata);
    } else {
        snprintf(out, (size_t)cap, "ambience.lua");
    }
#elif defined(__APPLE__)
    const char *xdg = getenv("XDG_CONFIG_HOME");
    if (xdg != NULL && xdg[0] != '\0') {
        snprintf(out, (size_t)cap, "%s/clanker/ambience.lua", xdg);
        return;
    }
    const char *h = helper();
    if (h != NULL) {
        snprintf(out, (size_t)cap, "%s/Library/Application Support/clanker/ambience.lua", h);
    } else {
        snprintf(out, (size_t)cap, "ambience.lua");
    }
#else
    const char *xdg = getenv("XDG_CONFIG_HOME");
    if (xdg != NULL && xdg[0] != '\0') {
        snprintf(out, (size_t)cap, "%s/clanker/ambience.lua", xdg);
        return;
    }
    const char *h = helper();
    if (h != NULL) {
        snprintf(out, (size_t)cap, "%s/.config/clanker/ambience.lua", h);
    } else {
        snprintf(out, (size_t)cap, "ambience.lua");
    }
#endif
}
