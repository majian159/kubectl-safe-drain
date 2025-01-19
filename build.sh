#!/usr/bin/env bash

# Define OS and architectures
osArray=("linux" "darwin" "freebsd" "windows")
archs=("amd64" "386" "arm64")

# If you want to build only for darwin on amd64 and arm64, you can adjust archs for darwin
archsForDarwin=("amd64" "arm64")

# Version
version=${1-"0.0.1-preview2"}
out_file="kubectl-safe-drain"

# Build function
build() {
  os=$1
  arch=$2

  GOOS=$os GOARCH=$arch go build -o "${out_file}" cmd/kubectl-sdrain.go
  tgzName="kubectl-safe-drain_${version}_${os}_${arch}.tar.gz"
  rm -f "${tgzName}"
  tar -czf "${tgzName}" kubectl-safe-drain
  rm -f "${out_file}"

  echo $(shasum -a 256 "${tgzName}")
}

# Main function
main() {
  for os in "${osArray[@]}"; do
    # For darwin, use a specific set of archs
    if [ "$os" == "darwin" ]; then
      # Darwin (macOS) needs both amd64 and arm64
      archsToUse=("${archsForDarwin[@]}")
    else
      # For other OS, use the default archs
      archsToUse=("${archs[@]}")
    fi

    # Loop through the selected architectures and build
    for arch in "${archsToUse[@]}"; do
      build "${os}" "${arch}"
    done
  done
}

# Run the main function
main