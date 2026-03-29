#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SVG_DIR="${ROOT_DIR}/assets/svg"
ASSETS_DIR="${ROOT_DIR}/assets"
ANDROID_RES_DIR="${ROOT_DIR}/android/app/src/main/res"

render_svg() {
  local input="$1"
  local size="$2"
  local output="$3"
  mkdir -p "$(dirname "${output}")"
  rsvg-convert "${input}" -w "${size}" -h "${size}" -o "${output}"
}

render_round_icon() {
  local input="$1"
  local size="$2"
  local output="$3"
  mkdir -p "$(dirname "${output}")"
  magick "${input}" \
    \( -size "${size}x${size}" xc:none -fill white -draw "circle $((size / 2)),$((size / 2)) $((size / 2)),$((size - 2))" \) \
    -alpha off -compose CopyOpacity -composite "${output}"
}

render_svg "${SVG_DIR}/icon-master.svg" 1024 "${ASSETS_DIR}/icon.png"
render_svg "${SVG_DIR}/adaptive-background.svg" 512 "${ASSETS_DIR}/android-icon-background.png"
render_svg "${SVG_DIR}/adaptive-foreground.svg" 512 "${ASSETS_DIR}/android-icon-foreground.png"
render_svg "${SVG_DIR}/monochrome.svg" 432 "${ASSETS_DIR}/android-icon-monochrome.png"
render_svg "${SVG_DIR}/splash.svg" 1024 "${ASSETS_DIR}/splash-icon.png"
render_svg "${SVG_DIR}/icon-master.svg" 48 "${ASSETS_DIR}/favicon.png"

for density in mdpi hdpi xhdpi xxhdpi xxxhdpi; do
  case "${density}" in
    mdpi) full_size=48; adaptive_size=108; splash_size=288 ;;
    hdpi) full_size=72; adaptive_size=162; splash_size=432 ;;
    xhdpi) full_size=96; adaptive_size=216; splash_size=576 ;;
    xxhdpi) full_size=144; adaptive_size=324; splash_size=864 ;;
    xxxhdpi) full_size=192; adaptive_size=432; splash_size=1152 ;;
  esac

  full_icon_path="${ANDROID_RES_DIR}/mipmap-${density}/ic_launcher.png"
  render_svg "${SVG_DIR}/icon-master.svg" "${full_size}" "${full_icon_path}"
  render_round_icon "${full_icon_path}" "${full_size}" "${ANDROID_RES_DIR}/mipmap-${density}/ic_launcher_round.png"

  render_svg "${SVG_DIR}/adaptive-background.svg" "${adaptive_size}" "${ANDROID_RES_DIR}/mipmap-${density}/ic_launcher_background.png"
  render_svg "${SVG_DIR}/adaptive-foreground.svg" "${adaptive_size}" "${ANDROID_RES_DIR}/mipmap-${density}/ic_launcher_foreground.png"
  render_svg "${SVG_DIR}/monochrome.svg" "${adaptive_size}" "${ANDROID_RES_DIR}/mipmap-${density}/ic_launcher_monochrome.png"

  render_svg "${SVG_DIR}/splash.svg" "${splash_size}" "${ANDROID_RES_DIR}/drawable-${density}/splashscreen_logo.png"
done
