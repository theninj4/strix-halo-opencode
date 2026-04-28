#!/bin/bash
set -ex
cd ./llama.cpp/

rm -rf build/CMakeFiles build/CMakeCache.txt
git clean -xdf && git pull && git submodule update --recursive

sudo pacman -S --needed \
  base-devel cmake ninja git curl \
  vulkan-icd-loader vulkan-headers \
  vulkan-radeon spirv-headers \
  shaderc spirv-tools glslang \
  vulkan-tools

cmake -S . -B build \
  -DGGML_VULKAN=ON \
  -DGGML_NATIVE=ON \
  -DCMAKE_BUILD_TYPE=Release \
  -DGGML_RPC=ON \
  -DCMAKE_INSTALL_PREFIX=/usr \
  -DLLAMA_BUILD_TESTS=OFF \
  -DLLAMA_BUILD_EXAMPLES=ON \
  -DLLAMA_BUILD_SERVER=ON \
  && cmake --build build --config Release -- -j 16
