---
title: Release Process
description: How releases are created for the Argo Trading framework using GitHub Actions
---

# Release Process

This document describes how releases are created for the Argo Trading framework.

## Overview

Releases are created through a **manual GitHub Actions workflow** that:
1. Determines the next version using semantic versioning
2. Builds platform-specific artifacts (macOS, Linux, Swift)
3. Signs and notarizes macOS binaries
4. Creates a GitHub release with all artifacts

## Triggering a Release

Releases are triggered manually via the GitHub Actions UI:

1. Navigate to **Actions** → **Create Release**
2. Click **Run workflow**
3. Select the `main` branch
4. Click **Run workflow**

The workflow is defined in [`.github/workflows/create-release.yaml`](../.github/workflows/create-release.yaml).

## Versioning

The project uses [Semantic Versioning](https://semver.org/) with the format `v{MAJOR}.{MINOR}.{PATCH}`.

Version bumps are determined automatically by analyzing commit messages using [semantic-release](https://semantic-release.gitbook.io/):

| Commit Prefix | Version Bump | Example |
|---------------|--------------|---------|
| `feat:` | Minor (1.x.0) | `feat: add new indicator` |
| `fix:` | Patch (1.0.x) | `fix: correct RSI calculation` |
| `BREAKING CHANGE` | Major (x.0.0) | Breaking change in commit body |

## Release Workflow Steps

The release workflow runs 5 sequential jobs:

### 1. Get Version (`get-version`)

- Runs `semantic-release` in dry-run mode
- Analyzes commits since the last release
- Determines the next version number
- Generates release notes from commit messages

### 2. Build XCFramework (`build-xcframework`)

- Builds Swift framework using `gomobile bind`
- Signs the framework with Apple developer certificate
- Creates a checksummed zip file
- Computes SHA256 checksum for Swift Package Manager

### 3. Build macOS Binaries (`build-macos-binaries`)

- Builds CLI tools: `trading-backtest` and `trading-market`
- Signs binaries with hardened runtime
- Creates installer package (`.pkg`)
- Notarizes the package with Apple's notary service
- Staples notarization ticket

### 4. Build Linux Binaries (`build-linux-binaries`)

- Builds `trading-backtest` for Linux (amd64)
- Packages binary in a zip file

### 5. Create Release (`create-release`)

- Downloads all build artifacts
- Updates `Package.swift` with new version URL and checksum
- Commits and pushes changes to `main`
- Creates annotated Git tag
- Publishes GitHub Release with artifacts

## Published Artifacts

Each release publishes three artifacts:

| Artifact | Platform | Description |
|----------|----------|-------------|
| `ArgoTrading.xcframework.zip` | macOS/iOS | Swift framework for integration |
| `ArgoTrading_macOS_arm64.pkg` | macOS | Installer package with CLI tools |
| `ArgoTrading_Linux_amd64.zip` | Linux | CLI binary archive |

## Package.swift Integration

The Swift package manifest is automatically updated during release:

```swift
.binaryTarget(
    name: "ArgoTrading",
    url: "https://github.com/rxtech-lab/argo-trading/releases/download/v{VERSION}/ArgoTrading.xcframework.zip",
    checksum: "{SHA256_CHECKSUM}"
)
```

Swift Package Manager users receive the new version automatically.

## Code Signing & Notarization

All macOS artifacts are signed and notarized:

- **Binary Signing**: Hardened runtime with entitlements
- **Framework Signing**: Developer certificate (inside-out signing)
- **Installer Signing**: Installer certificate
- **Notarization**: Required for macOS distribution outside App Store

## Required GitHub Secrets

The release workflow requires these secrets to be configured:

| Secret | Purpose |
|--------|---------|
| `RELEASE_TOKEN` | Push commits and create releases |
| `BUILD_CERTIFICATE_BASE64` | Apple developer certificate |
| `P12_PASSWORD` | Certificate password |
| `SIGNING_CERTIFICATE_NAME` | Developer certificate ID |
| `INSTALLER_CERTIFICATE_BASE64` | Installer certificate |
| `INSTALLER_SIGNING_CERTIFICATE_NAME` | Installer certificate ID |
| `APPLE_ID` | Apple ID for notarization |
| `APPLE_ID_PWD` | App-specific password |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `DEPLOY_KEY` | SSH key for repository push |

## CI vs Release

The CI workflow ([`.github/workflows/ci.yml`](../.github/workflows/ci.yml)) runs on every PR and push to `main`:
- Runs tests with race detection
- Uploads coverage to Codecov
- Compiles WASM examples
- Runs linting

The CI workflow does **not** publish releases. Use the release workflow for production releases.
