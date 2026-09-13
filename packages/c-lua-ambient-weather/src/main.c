// main.c — transparent fullscreen weather overlay.
//
// Modes:
//   ./app                        run the SDL overlay (default)
//   ./app --once                  fetch weather, print config+weather, exit
//   ./app --config path           use an explicit Lua config file
//   ./app --location "Berlin"     override location for this run
//   ./app --help                  usage
//
// The overlay window is borderless, transparent, always-on-top (configurable)
// and click-through so it "shades" the desktop without stealing input.
// Quit with q / Esc or by closing the window.
#include <SDL3/SDL.h>
#include <curl/curl.h>

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "config.h"
#include "fx.h"
#include "platform.h"
#include "weather.h"

static void usage(const char *argv0) {
    printf("usage: %s [options]\n", argv0);
    printf("\n");
    printf("  --once               print config + weather and exit (no window)\n");
    printf("  --config PATH        load this Lua file instead of the defaults chain\n");
    printf("  --location NAME      override location (e.g. \"Berlin, Germany\")\n");
    printf("  --coords LAT LON     override with explicit coordinates\n");
    printf("  --help               this text\n");
    printf("\n");
    printf("Config files: <global> then ./ambience.lua (local wins).\n");
    printf("  Linux   : $XDG_CONFIG_HOME/clanker/ambience.lua or ~/.config/clanker/ambience.lua\n");
    printf("  macOS   : ~/Library/Application Support/clanker/ambience.lua\n");
    printf("  Windows : %%APPDATA%%\\clanker\\ambience.lua\n");
}

static bool has_display(void) {
#ifdef _WIN32
    return true;
#else
    return getenv("DISPLAY") != NULL || getenv("WAYLAND_DISPLAY") != NULL;
#endif
}

int main(int argc, char **argv) {
    // Make sure we actually got valid arguments.
    if (argv == NULL || argc < 1) {
        fprintf(stderr, "ambience: invalid command line arguments\n");
        return 2;
    }
    const char *explicit_config = NULL;
    const char *location_override = NULL;
    double override_lat = 0, override_lon = 0;
    bool has_override_coords = false;
    bool once = false;

    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--help") == 0 || strcmp(argv[i], "-h") == 0) {
            usage(argv[0]);
            return 0;
        } else if (strcmp(argv[i], "--once") == 0) {
            once = true;
        } else if (strcmp(argv[i], "--config") == 0) {
            if (i + 1 >= argc) {
                fprintf(stderr, "ambience: --config needs a PATH argument\n");
                usage(argv[0]);
                return 2;
            }
            explicit_config = argv[++i];
        } else if (strcmp(argv[i], "--location") == 0) {
            if (i + 1 >= argc) {
                fprintf(stderr, "ambience: --location needs a NAME argument\n");
                usage(argv[0]);
                return 2;
            }
            location_override = argv[++i];
        } else if (strcmp(argv[i], "--coords") == 0) {
            if (i + 2 >= argc) {
                fprintf(stderr, "ambience: --coords needs LAT and LON arguments\n");
                usage(argv[0]);
                return 2;
            }
            override_lat = atof(argv[++i]);
            override_lon = atof(argv[++i]);
            has_override_coords = true;
        } else {
            fprintf(stderr, "unknown argument: %s\n", argv[i]);
            usage(argv[0]);
            return 2;
        }
    }

    Config cfg;
    int rc;
    if (explicit_config != NULL) {
        config_defaults(&cfg);
        rc = config_load_file(&cfg, explicit_config);
        if (rc != 0) {
            return 1;
        }
    } else {
        rc = config_load(&cfg);
        if (rc != 0) {
            return 1;
        }
    }
    if (location_override != NULL) {
        snprintf(cfg.location, sizeof cfg.location, "%s", location_override);
        cfg.has_coords = false;
    }
    if (has_override_coords) {
        cfg.latitude = override_lat;
        cfg.longitude = override_lon;
        cfg.has_coords = true;
    }

    curl_global_init(CURL_GLOBAL_DEFAULT);

    Weather w = { 0 };
    if (weather_fetch(&cfg, &w) != 0) {
        fprintf(stderr, "ambience: continuing with no weather (offline?); retrying in loop\n");
    }

    config_print(&cfg);
    weather_print(&w);
    fflush(stdout);

    if (once) {
        float ri = weather_rain_intensity(&w, &cfg);
        float si = weather_snow_intensity(&w, &cfg);
        float wi = weather_wind_intensity(&w, &cfg);
        printf("fx intensities: rain=%.2f snow=%.2f wind=%.2f cloud=%.2f fog=%.2f thunder=%d\n", ri,
               si, wi, weather_cloud_factor(&w), weather_fog_factor(&w), weather_is_thunder(&w));
        curl_global_cleanup();
        return w.valid ? 0 : 1;
    }

    if (!has_display()) {
        fprintf(stderr, "ambience: no display found (DISPLAY/WAYLAND_DISPLAY unset).\n"
                        "Use --once for headless weather/config check.\n");
        curl_global_cleanup();
        return 3;
    }

    if (!SDL_Init(SDL_INIT_VIDEO)) {
        fprintf(stderr, "ambience: SDL_Init failed: %s\n", SDL_GetError());
        curl_global_cleanup();
        return 1;
    }

    SDL_DisplayID disp = SDL_GetPrimaryDisplay();
    SDL_Rect bounds = { 0, 0, 1280, 720 };
    if (disp != 0) {
        SDL_GetDisplayBounds(disp, &bounds);
    }

    SDL_WindowFlags flags = SDL_WINDOW_BORDERLESS | SDL_WINDOW_TRANSPARENT | SDL_WINDOW_RESIZABLE;
    if (cfg.window.always_on_top) {
        flags |= SDL_WINDOW_ALWAYS_ON_TOP;
    }
    flags |= SDL_WINDOW_NOT_FOCUSABLE;

    SDL_Window *win =
        SDL_CreateWindow("ambience", bounds.w > 0 ? bounds.w : 1280, bounds.h > 0 ? bounds.h : 720, flags);
    if (win == NULL) {
        fprintf(stderr, "ambience: SDL_CreateWindow failed: %s\n", SDL_GetError());
        SDL_Quit();
        curl_global_cleanup();
        return 1;
    }
    SDL_SetWindowPosition(win, bounds.x, bounds.y);
    SDL_SetWindowFullscreen(win, true);
    if (cfg.window.click_through) {
        platform_click_through(win); // best effort; NOT_FOCUSABLE always set
    }
    SDL_SetWindowOpacity(win, cfg.window.opacity);
    SDL_SetWindowTitle(win, "ambience");

    SDL_Renderer *ren = SDL_CreateRenderer(win, NULL);
    if (ren == NULL) {
        fprintf(stderr, "ambience: SDL_CreateRenderer failed: %s\n", SDL_GetError());
        SDL_DestroyWindow(win);
        SDL_Quit();
        curl_global_cleanup();
        return 1;
    }
    SDL_SetRenderDrawBlendMode(ren, SDL_BLENDMODE_BLEND);
    SDL_SetRenderVSync(ren, 1);

    Fx fx;
    int ww = 0, wh = 0;
    SDL_GetWindowSize(win, &ww, &wh);
    fx_init(&fx, ww, wh);

    time_t time_value = time(NULL) + cfg.update_interval;
    uint64_t last = SDL_GetTicks();
    bool running = true;

    while (running) {
        // Poll and handle all pending SDL events for this frame.
        SDL_Event ev;
        while (SDL_PollEvent(&ev)) {
            if (ev.type == SDL_EVENT_QUIT) {
                running = false;
            } else if (ev.type == SDL_EVENT_KEY_DOWN) {
                SDL_Keycode k = ev.key.key;
                if (k == SDLK_Q || k == SDLK_ESCAPE) {
                    running = false;
                }
            } else if (ev.type == SDL_EVENT_WINDOW_RESIZED || ev.type == SDL_EVENT_WINDOW_PIXEL_SIZE_CHANGED) {
                int nw = 0, nh = 0;
                SDL_GetWindowSize(win, &nw, &nh);
                fx_resize(&fx, nw, nh);
            }
        }

        time_t now = time(NULL);
        if (now >= time_value) {
            Weather nw = { 0 };
            if (weather_fetch(&cfg, &nw) == 0) {
                w = nw;
                weather_print(&w);
                fflush(stdout);
            } else {
                fprintf(stderr, "ambience: refetch failed, keeping old weather\n");
            }
            time_value = now + cfg.update_interval;
        }

        uint64_t cur = SDL_GetTicks();
        // Convert the millisecond delta into a float delta in seconds.
        float dt = ((float)(cur - last)) / 1000.0f;
        last = cur;
        if (dt < 0.0f) {
            dt = 0.016f;
        }
        if (dt > 0.1f) {
            dt = 0.1f;
        }

        fx_update(&fx, &cfg, &w, dt);

        // Clear the screen to fully transparent, then draw the effects on top.
        SDL_SetRenderDrawColor(ren, 0, 0, 0, 0);
        SDL_RenderClear(ren);
        fx_render(ren, &fx, &cfg, &w);
        SDL_RenderPresent(ren); // present the finished frame to the window
    }

    SDL_DestroyRenderer(ren);
    SDL_DestroyWindow(win);
    SDL_Quit();
    curl_global_cleanup();
    return 0;
}
