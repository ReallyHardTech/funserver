# Installer Assets

This directory contains assets for creating installation packages with GoReleaser.

## Files Required for Windows MSI Installer

- `icon.ico` - The application icon for Windows (48x48, 32x32, and 16x16 sizes)
- `LICENSE.rtf` - RTF formatted license file for the Windows installer

## Files Required for macOS DMG Installer

- `icon.icns` - The application icon for macOS
- `dmg-background.png` - Background image for the DMG installer window (recommended size: 540x380)

## How to Create These Files

### Windows Icon (.ico)

1. Create PNG images in the following sizes: 16x16, 32x32, 48x48
2. Use an online converter or tool like ImageMagick to combine them into an .ico file

### macOS Icon (.icns)

1. Create a 1024x1024 PNG image
2. Use `iconutil` on macOS or a third-party tool to convert it to .icns format

### DMG Background

1. Create a PNG image with recommended dimensions of 540x380 pixels
2. Use subtle visual cues to indicate where users should drag the application

### License RTF

1. Convert the LICENSE file to RTF format using a word processor or online converter 