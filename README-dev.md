# Dev Readme
- [Dev Readme](#dev-readme)
  - [Overview](#overview)
  - [Initial Setup](#initial-setup)
  - [Build and Run](#build-and-run)
  - [Releases](#releases)

## Overview
- dev notes for later reference and to save time

## Initial Setup
```bash
# init go module
go mod init github.com/adegoodyer/twingate-connector-manager && \
go mod tidy

# local install
go install twingate-connector-manager.go

# run
twingate-connector-manager
```

## Build and Run
```bash
# build binary
go build -o bin/twingate-connector-manager ./cmd/twingate-connector-manager

# run binary
./bin/twingate-connector-manager
```

## Releases
```bash
# tag release version and latest
# removes existing latest tag if it exists
# creates new release version and latest tag
export RELEASE=v0.0.1 && \
git tag -d latest 2>/dev/null && \
git push origin --delete latest 2>/dev/null || true && \
git tag -a $RELEASE -m "Release version $RELEASE" && \
git push origin $RELEASE && \
git tag -a latest -m "Latest release" && \
git push origin latest

# test remote install
go install github.com/adegoodyer/twingate-connector-manager@latest

# run app
twingate-connector-manager
```
