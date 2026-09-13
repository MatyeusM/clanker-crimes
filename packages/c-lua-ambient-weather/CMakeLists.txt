cmake_minimum_required(VERSION 4.4.3)

project(clanker-weather-ambience
    VERSION 0.1.0
    LANGUAGES C
)

set(CMAKE_C_STANDARD 23)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_C_EXTENSIONS OFF)

include(FetchContent)
# lua/lua has no CMakeLists.txt; populate sources manually (needs OLD policy
# on CMake >= 3.30 where FetchContent_Populate is deprecated).
if(POLICY CMP0169)
    cmake_policy(SET CMP0169 OLD)
endif()

FetchContent_Declare(
    SDL3
    GIT_REPOSITORY https://github.com/libsdl-org/SDL.git
    GIT_TAG release-3.4.12
    GIT_SHALLOW TRUE
)

FetchContent_Declare(
    yyjson
    GIT_REPOSITORY https://github.com/ibireme/yyjson.git
    GIT_TAG 0.13.0
    GIT_SHALLOW TRUE
)

FetchContent_Declare(
    curl
    GIT_REPOSITORY https://github.com/curl/curl.git
    GIT_TAG curl-8_22_0
    GIT_SHALLOW TRUE
)

# NOTE: upstream lua/lua has no CMakeLists.txt, so we only populate it
# and build a static `lua` target ourselves (no new third-party libs).
FetchContent_Declare(
    lua
    GIT_REPOSITORY https://github.com/lua/lua.git
    GIT_TAG v5.5.1
    GIT_SHALLOW TRUE
    CONFIGURE_COMMAND ""
    BUILD_COMMAND ""
)

FetchContent_MakeAvailable(SDL3 yyjson curl)
FetchContent_GetProperties(lua)
if(NOT lua_POPULATED)
    FetchContent_Populate(lua)
endif()

# --- Lua 5.5 static library from fetched sources ---------------------------
set(LUA_CORE_SOURCES
    ${lua_SOURCE_DIR}/lapi.c
    ${lua_SOURCE_DIR}/lauxlib.c
    ${lua_SOURCE_DIR}/lbaselib.c
    ${lua_SOURCE_DIR}/lcode.c
    ${lua_SOURCE_DIR}/lcorolib.c
    ${lua_SOURCE_DIR}/lctype.c
    ${lua_SOURCE_DIR}/ldblib.c
    ${lua_SOURCE_DIR}/ldebug.c
    ${lua_SOURCE_DIR}/ldo.c
    ${lua_SOURCE_DIR}/ldump.c
    ${lua_SOURCE_DIR}/lfunc.c
    ${lua_SOURCE_DIR}/lgc.c
    ${lua_SOURCE_DIR}/linit.c
    ${lua_SOURCE_DIR}/liolib.c
    ${lua_SOURCE_DIR}/llex.c
    ${lua_SOURCE_DIR}/lmathlib.c
    ${lua_SOURCE_DIR}/lmem.c
    ${lua_SOURCE_DIR}/loadlib.c
    ${lua_SOURCE_DIR}/lobject.c
    ${lua_SOURCE_DIR}/lopcodes.c
    ${lua_SOURCE_DIR}/loslib.c
    ${lua_SOURCE_DIR}/lparser.c
    ${lua_SOURCE_DIR}/lstate.c
    ${lua_SOURCE_DIR}/lstring.c
    ${lua_SOURCE_DIR}/lstrlib.c
    ${lua_SOURCE_DIR}/ltable.c
    ${lua_SOURCE_DIR}/ltablib.c
    ${lua_SOURCE_DIR}/ltm.c
    ${lua_SOURCE_DIR}/lundump.c
    ${lua_SOURCE_DIR}/lutf8lib.c
    ${lua_SOURCE_DIR}/lvm.c
    ${lua_SOURCE_DIR}/lzio.c
)

add_library(lua STATIC ${LUA_CORE_SOURCES})
target_include_directories(lua PUBLIC ${lua_SOURCE_DIR})
# loadlib needs dl on POSIX; harmless elsewhere
find_library(DL_LIB dl)
if(DL_LIB)
    target_link_libraries(lua PUBLIC ${DL_LIB})
endif()
if(UNIX)
    target_link_libraries(lua PUBLIC m)
endif()

# --- App -------------------------------------------------------------------
add_executable(clanker-weather-ambience
    src/main.c
    src/config_defaults.c
    src/config_path.c
    src/config_lua.c
    src/config_apply.c
    src/config_push.c
    src/config_load.c
    src/http.c
    src/weather.c
    src/weather_eval.c
    src/fx.c
    src/fx_render.c
    src/platform.c
)

target_include_directories(clanker-weather-ambience PRIVATE src)

target_link_libraries(clanker-weather-ambience
    PRIVATE
        SDL3::SDL3
        yyjson
        CURL::libcurl
        lua
)

# platform.c extras (system libs only, no new third-party deps):
# - macOS: Cocoa for NSWindow setIgnoresMouseEvents (click-through)
# - X11 desktops: Xext Shape extension for an empty input shape
if(APPLE)
    target_link_libraries(clanker-weather-ambience PRIVATE "-framework Cocoa")
elseif(UNIX)
    find_package(X11)
    if(X11_FOUND AND X11_Xext_FOUND)
        target_include_directories(clanker-weather-ambience PRIVATE ${X11_INCLUDE_DIR})
        target_link_libraries(clanker-weather-ambience PRIVATE ${X11_LIBRARIES} ${X11_Xext_LIB})
    endif()
endif()

if(MSVC)
    target_compile_options(clanker-weather-ambience PRIVATE /W4)
else()
    target_compile_options(clanker-weather-ambience PRIVATE -Wall -Wextra -Wpedantic)
endif()

install(TARGETS clanker-weather-ambience RUNTIME DESTINATION bin)
