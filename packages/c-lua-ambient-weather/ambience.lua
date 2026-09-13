-- ambience.lua — example config for clanker-weather-ambience.
--
-- Load order: built-in defaults <- global config <- ./ambience.lua (this file wins).
--   Linux   : $XDG_CONFIG_HOME/clanker/ambience.lua or ~/.config/clanker/ambience.lua
--   macOS   : ~/Library/Application Support/clanker/ambience.lua
--   Windows : %APPDATA%\clanker\ambience.lua
--   Local   : ./ambience.lua in the working directory (overrides global)
--
-- Every key is optional. Only set what you want to change.
-- Colors are { r=0..255, g=0..255, b=0..255, a=0..255 } (array form {r,g,b,a} works too).

-- Place name resolved via Open-Meteo geocoding (no API key needed).
-- Alternatively set latitude/longitude below to skip geocoding.
location = "Nice, France"
-- latitude = 43.70
-- longitude = 7.27

units = "metric"        -- "metric" (C, km/h) | "imperial" (F, mph)
update_interval = 600   -- seconds between weather refetches (min 30)

window = {
  opacity = 1.0,        -- 0..1 window opacity hint
  click_through = true, -- mouse passthrough so the overlay never steals clicks
  always_on_top = true, -- keep the shade above the wallpaper / below panels per WM
}

-- Night: dark blue overlay while Open-Meteo reports is_day=0.
night = {
  enabled = true,
  color = { r = 4, g = 8, b = 28 },
  max_alpha = 110,      -- 0..255 peak strength at night
}

-- Temperature tint: cold blue below cold_threshold, hot orange above hot_threshold.
temperature = {
  enabled = true,
  cold_threshold = 5.0,
  hot_threshold = 28.0,
  cold_color = { r = 90, g = 140, b = 255 },
  hot_color = { r = 255, g = 130, b = 40 },
  max_alpha = 64,
}

-- Clouds: gray dimming proportional to cloud_cover %.
clouds = {
  enabled = true,
  color = { r = 110, g = 120, b = 140 },
  max_alpha = 90,       -- alpha at 100% cover
}

-- Fog: drifting translucent puffs, driven by low visibility / humidity / fog code.
fog = {
  enabled = true,
  color = { r = 200, g = 210, b = 220 },
  max_alpha = 80,
  count = 24,           -- max puffs (0..256)
  speed = 12.0,         -- px/s drift
}

-- Rain: slanted line drops, intensity from rain+showers mm/h.
rain = {
  enabled = true,
  color = { r = 140, g = 170, b = 255, a = 140 },
  count = 900,          -- max drops (0..4000)
  length = 18.0,        -- px
  speed = 900.0,        -- px/s fall speed
  full_rate = 4.0,      -- mm/h that gives intensity 1.0
  intensity_scale = 1.0,-- global multiplier (try 0.5 for subtle, 1.5 for heavy)
}

-- Snow: drifting dots, intensity from snowfall cm/h.
snow = {
  enabled = true,
  color = { r = 255, g = 255, b = 255, a = 220 },
  count = 700,          -- max flakes (0..4000)
  size_min = 1.5,
  size_max = 3.5,
  speed = 70.0,         -- px/s fall speed
  drift = 40.0,         -- px/s sway amplitude
  full_rate = 1.0,      -- cm/h that gives intensity 1.0
  intensity_scale = 1.0,
}

-- Wind: horizontal streaks, driven by wind_speed_10m.
wind = {
  enabled = true,
  color = { r = 255, g = 255, b = 255, a = 90 },
  count = 120,          -- max streaks (0..1000)
  speed_scale = 8.0,    -- px/s gained per km/h (or mph in imperial)
  min_speed = 10.0,     -- streaks start appearing here
  full_speed = 40.0,    -- intensity 1.0 here
  streak_length = 60.0, -- px
  intensity_scale = 1.0,
}

-- Storm: fullscreen flash on lightning during thunderstorm codes (95/96/99).
storm = {
  enabled = true,
  color = { r = 255, g = 255, b = 255 },
  flash_alpha = 160,    -- peak flash strength
  bolts_per_min = 4.0,  -- average rate while thunder is active
}
