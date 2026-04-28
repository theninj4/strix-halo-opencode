
# Running OpenCode on Strix Halo

*Hardware*: AMD Ryzen AI MAX+ 395 64GB+
```
| model                   |       size |   params | backend  |            test |            t/s |
| ----------------------- | ---------: | -------: | -------- | --------------: |--------------: |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |           pp256 | 736.72 ± 12.48 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |            tg64 |   45.65 ± 0.16 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |   pp256 @ d8000 |  678.28 ± 4.69 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |    tg64 @ d8000 |   43.96 ± 0.14 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |  pp256 @ d16000 |  619.87 ± 8.11 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |   tg64 @ d16000 |   42.45 ± 0.08 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |  pp256 @ d32000 |  522.93 ± 7.16 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |   tg64 @ d32000 |   39.97 ± 0.03 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |  pp256 @ d64000 |  408.79 ± 1.86 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |   tg64 @ d64000 |   35.71 ± 0.10 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   | pp256 @ d128000 |  279.76 ± 1.47 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |  tg64 @ d128000 |   29.42 ± 0.14 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   | pp256 @ d256000 |  140.16 ± 8.28 |
| qwen35moe 35B.A3B Q8_0  |  35.80 GiB |  34.66 B | Vulkan   |  tg64 @ d256000 |   21.75 ± 0.12 |
```

## 1 - Install Omarchy (opinionated Arch Linux)

Head on over to https://omarchy.org/ download the ISO, install it.

## 2 - Setup the Kernel cmdline

Edit this file:
```bash
sudo nvim /boot/limine.conf
```

You're looking for around ~L29, something like this:

>   cmdline: quiet splash.....

You want to add these on the end:

> iommu=pt amdgpu.gttsize=126976 ttm.pages_limit=32505856

## 3 - Install the dependencies

```bash
sudo pacman -S --needed \
  base-devel cmake ninja git curl \
  vulkan-icd-loader vulkan-headers \
  vulkan-radeon spirv-headers \
  shaderc spirv-tools glslang \
  vulkan-tools
```

## 3 - Clone and build llama.cpp

```bash
mkdir -p ~/repos
cd ~/repos
git clone https://github.com/ggml-org/llama.cpp.git
cd ~/repos/llama.cpp
rm -rf build/CMakeFiles build/CMakeCache.txt
git clean -xdf && git pull && git submodule update --recursive
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
```

## 5 - Clone and build llama-runner

```bash
mkdir -p ~/repos
cd ~/repos
git clone https://github.com/theninj4/strix-halo-opencode.git
cd ~/repos/strix-halo-opencode
GOOS=linux GOARCH=amd64 go build -o llama-runner ./strix-halo-opencode/llama-runner/...
```

## 6 - Download the Qwen weights

```bash
mkdir -p ~/repos/strix-halo-opencode/models
uvx hf download --local-dir ~/repos/strix-halo-opencode/models unsloth/Qwen3.6-35B-A3B-GGUF Qwen3.6-35B-A3B-UD-Q8_K_XL.gguf
```

## 7 - Start the server

```bash
~/repos/strix-halo-opencode/llama-runner --server ~/repos/llama.cpp/build/bin/llama-server --config ~/repos/strix-halo-opencode/presets_qwen.ini
```

## 8 - Setup OpenCode

Follow the instructions on their website:
https://opencode.ai/
then grab our config:

```bash
cp ~/repos/strix-halo-opencode/opencode.json ~/.config/opencode/opencode.json
```

Finally, replace the `__IP_ADDRESS_OF_STRIX_HALO__` marker with the IP address of your machine!
