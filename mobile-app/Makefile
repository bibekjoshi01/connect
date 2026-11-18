SHELL := /bin/bash
FLUTTER ?= flutter
DART ?= dart

.PHONY: help deps run check apk format

help:
	@echo 'Android app only: make deps, run [ARGS=...], check, apk, format.'
	@echo 'Requires Flutter 3.47.8; Android builds also need JDK 21 and the Android SDK.'

deps:
	"$(FLUTTER)" pub get --enforce-lockfile

run: deps
	"$(FLUTTER)" run $(ARGS)

check: deps
	"$(DART)" format --output=none --set-exit-if-changed lib test
	"$(FLUTTER)" analyze --no-pub --fatal-infos
	"$(FLUTTER)" test --no-pub

apk: deps
	"$(FLUTTER)" build apk --debug --no-pub

format:
	"$(DART)" format lib test
