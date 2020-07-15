#!/usr/bin/env bash
osArray=("linux" "darwin" "freebsd" "windows")
archs=("amd64" "386")
version=${1-"0.0.1-preview1"}
out_file="kubectl-safe-drain"

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

main() {
  for os in "${osArray[@]}"; do
    for arch in "${archs[@]}"; do
      build "${os}" "${arch}"
    done
  done
}

main
