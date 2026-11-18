SHELL := /bin/bash
CLANG_FORMAT ?= clang-format
ROOT_DIR := $(shell cd .. && pwd)
BUILD_DIR := $(ROOT_DIR)/.build/native-engine
CPP_FILES := $(shell find include src tests -type f \( -name '*.cpp' -o -name '*.hpp' \))

.PHONY: help build test check format format-check

help:
	@echo 'Portable C++20 library only: make build, test, check, format, format-check.'
	@echo 'CMake can also build this project directly; Flutter and Go are not required.'

build:
	cmake -S . -B "$(BUILD_DIR)" -DCMAKE_BUILD_TYPE=Debug -DBUILD_TESTING=ON
	cmake --build "$(BUILD_DIR)" --parallel

test: build
	ctest --test-dir "$(BUILD_DIR)" --output-on-failure

format:
	"$(CLANG_FORMAT)" -i $(CPP_FILES)

format-check:
	"$(CLANG_FORMAT)" --dry-run --Werror $(CPP_FILES)

check: format-check test
