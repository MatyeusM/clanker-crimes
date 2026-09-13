// platform.h — best-effort native click-through for the overlay window.
//
// SDL gives us a borderless always-on-top transparent window, but true
// mouse passthrough is platform-specific. Wayland has no client-side API
// for it (window stays NOT_FOCUSABLE there instead).
#pragma once

#include <SDL3/SDL.h>

// Makes clicks fall through to the desktop. No-op where unsupported.
void platform_click_through(SDL_Window *win);
