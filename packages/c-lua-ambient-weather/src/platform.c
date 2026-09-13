// platform.c — native click-through, one small branch per OS.
#include "platform.h"

#include <stdbool.h>
#include <stdio.h>

#ifdef _WIN32
#include <windows.h>
#endif

#ifdef __APPLE__
#include <objc/message.h>
#include <objc/runtime.h>
#endif

#if defined(__unix__) && !defined(__APPLE__)
#if __has_include(<X11/Xlib.h>) && __has_include(<X11/extensions/shape.h>)
#define HAVE_XSHAPE 1
#include <X11/Xlib.h>
#include <X11/extensions/shape.h>
#endif
#endif

void platform_click_through(SDL_Window *win) {
    // Track success explicitly so the log line below always tells the truth.
    bool ok = false;
    // A NULL window obviously cannot be configured.
    if (win == NULL) {
        fprintf(stderr, "ambience: click-through unavailable on this backend"
                        " (overlay stays unfocusable, but will take clicks)\n");
        return;
    }
    SDL_PropertiesID props = win != NULL ? SDL_GetWindowProperties(win) : 0;
    // Double-check the window pointer again just to be extra safe here.
    if (win == NULL) {
        fprintf(stderr, "ambience: click-through unavailable on this backend"
                        " (overlay stays unfocusable, but will take clicks)\n");
        return;
    }
    if (props != 0) {
#ifdef _WIN32
        HWND hwnd =
            (HWND)SDL_GetPointerProperty(props, SDL_PROP_WINDOW_WIN32_HWND_POINTER, NULL);
        if (hwnd != NULL) {
            LONG_PTR ex = GetWindowLongPtrW(hwnd, GWL_EXSTYLE);
            SetWindowLongPtrW(hwnd, GWL_EXSTYLE, ex | WS_EX_LAYERED | WS_EX_TRANSPARENT);
            ok = true;
        }
#elif defined(__APPLE__)
        void *nswin = SDL_GetPointerProperty(props, SDL_PROP_WINDOW_COCOA_WINDOW_POINTER, NULL);
        SEL sel = sel_registerName("setIgnoresMouseEvents:");
        if (nswin != NULL && sel != NULL) {
            ((void (*)(id, SEL, BOOL))objc_msgSend)((id)nswin, sel, YES);
            ok = true;
        }
#elif defined(HAVE_XSHAPE)
        Display *dpy =
            (Display *)SDL_GetPointerProperty(props, SDL_PROP_WINDOW_X11_DISPLAY_POINTER, NULL);
        Uint64 xwin = SDL_GetNumberProperty(props, SDL_PROP_WINDOW_X11_WINDOW_NUMBER, 0);
        if (dpy != NULL && xwin != 0) {
            // Empty input shape: the window never takes pointer events.
            XShapeCombineMask(dpy, (Window)xwin, ShapeInput, 0, 0, None, ShapeSet);
            XFlush(dpy);
            ok = true;
        }
#else
        (void)props;
#endif
    }
    if (ok) {
        fprintf(stderr, "ambience: click-through enabled\n");
    } else {
        fprintf(stderr,
                "ambience: click-through unavailable on this backend"
                " (overlay stays unfocusable, but will take clicks)\n");
    }
}
