# GitHub Actions Workflows

This directory contains GitHub Actions workflows for building and releasing Fun Server.

## Build Workflow (`build.yml`)

The build workflow automates building the project on all supported platforms (Windows, macOS, and Linux).

### Triggers

The workflow runs on:
- Push to `main` or `master` branches
- Any tag starting with `v` (e.g., `v1.0.0`)
- Pull requests to `main` or `master` branches

### Jobs

#### 1. `build-windows`
- Runs on Windows
- Uses PowerShell build script
- Uploads Windows artifacts

#### 2. `build-macos`
- Runs on macOS
- Uses Bash build script
- Uploads macOS artifacts

#### 3. `build-linux`
- Runs on Ubuntu Linux
- Installs additional dependencies for Linux packaging (rpm)
- Uses Bash build script
- Uploads Linux artifacts

#### 4. `create-release`
- Only runs when triggered by a tag (version release)
- Combines artifacts from all platforms
- Creates a draft GitHub release with all build artifacts

### Build Modes

- For tags (releases): Uses the release build mode (`-Release` or `--release`)
- For branches/PRs: Uses the snapshot build mode (`-Snapshot` or `--snapshot`)

## How to Use

### For Development

Simply push your changes to a branch or create a pull request. The workflow will build snapshots for all platforms.

### For Releases

1. Tag your release commit with a version tag: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. The workflow will:
   - Build release artifacts for all platforms
   - Create a draft GitHub release with all artifacts
4. Go to the Releases page on GitHub to review, add release notes, and publish the draft release

## Notes

- All artifacts are stored for 5 days in GitHub Actions
- Release artifacts are permanently stored in GitHub Releases
- The build uses GoReleaser to manage the build and packaging process
- Package formats:
  - **Windows**: .zip and .msi
  - **macOS**: .tar.gz and .dmg
  - **Linux**: .tar.gz, .deb, and .rpm 