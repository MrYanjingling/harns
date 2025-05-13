#!/usr/bin/env bash

iot::util::sortable_date() {
  date "+%Y%m%d-%H%M%S"
}

iot::util::host_os() {
  local host_os
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      echo "Unsupported host OS.  Must be Linux or Mac OS X."
      exit 1
      ;;
  esac
  echo "${host_os}"
}

iot::util::host_arch() {
  local host_arch
  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    aarch64*)
      host_arch=arm64
      ;;
    arm64*)
      host_arch=arm64
      ;;
    arm*)
      host_arch=arm
      ;;
    i?86*)
      host_arch=x86
      ;;
    s390x*)
      host_arch=s390x
      ;;
    ppc64le*)
      host_arch=ppc64le
      ;;
    *)
      echo "Unsupported host arch. Must be x86_64, 386, arm, arm64, s390x or ppc64le."
      exit 1
      ;;
  esac
  echo "${host_arch}"
}

# This figures out the host platform without relying on golang.  We need this as
# we don't want a golang install to be a prerequisite to building yet we need
# this info to figure out where the final binaries are placed.
iot::util::host_platform() {
  echo "$(iot::util::host_os)/$(iot::util::host_arch)"
}

# looks for $1 in well-known output locations for the platform ($2)
# $IOT_ROOT must be set
iot::util::find-binary-for-platform() {
  local -r lookfor="$1"
  local -r platform="$2"
  local locations=(
    "${IOT_ROOT}/_output/bin/${lookfor}"
    "${IOT_ROOT}/_output/dockerized/bin/${platform}/${lookfor}"
    "${IOT_ROOT}/_output/local/bin/${platform}/${lookfor}"
    "${IOT_ROOT}/platforms/${platform}/${lookfor}"
  )
  # if we're looking for the host platform, add local non-platform-qualified search paths
  if [[ "${platform}" = "$(iot::util::host_platform)" ]]; then
    locations+=(
      "${IOT_ROOT}/_output/local/go/bin/${lookfor}"
      "${IOT_ROOT}/_output/dockerized/go/bin/${lookfor}"
    );
  fi

  # List most recently-updated location.
  local -r bin=$( (ls -t "${locations[@]}" 2>/dev/null || true) | head -1 )

  if [[ -z "${bin}" ]]; then
    echo "Failed to find binary ${lookfor} for platform ${platform}"
    return 1
  fi

  echo -n "${bin}"
}

# looks for $1 in well-known output locations for the host platform
# $IOT_ROOT must be set
iot::util::find-binary() {
  iot::util::find-binary-for-platform "$1" "$(iot::util::host_platform)"
}
