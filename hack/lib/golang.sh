#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


readonly IOT_GO_PACKAGE=lightiot
readonly IOT_GOPATH="${IOT_OUTPUT}/go"

iot::check::env() {
    errors=()
    if [ -z $GOPATH ]; then
        errors+="GOPATH environment value not set"
    fi

    # check other env

    # check length of errors
    if [[ ${#errors[@]} -ne 0 ]] ; then
        local error
        for error in "${errors[@]}"; do
            echo "Error: "$error
        done
        exit 1
    fi
}

iot::golang::binaries_from_targets() {
  local target
  for target in "$@"; do
    # If the target starts with what looks like a domain name, assume it has a
    # fully-qualified package name rather than one that needs the Kubernetes
    # package prepended.
    if [[ "${target}" =~ ^([[:alnum:]]+".")+[[:alnum:]]+"/" ]]; then
      echo "${target}"
    else
      echo "${IOT_GO_PACKAGE}/${target}"
    fi
  done
}

# Asks golang what it thinks the host platform is. The go tool chain does some
# slightly different things when the target platform matches the host platform.
iot::golang::host_platform() {
  echo "$(go env GOHOSTOS)/$(go env GOHOSTARCH)"
}

iot::golang::sever_targets() {
  local targets=(
    cmd/iot-model-manager
    cmd/iot-data-broker
    cmd/iot-data-collector
    cmd/iot-data-query
    cmd/iot-event-manager
    cmd/iot-rule-manager
    cmd/iot-notification-manager
    cmd/iot-installation-manager
    cmd/iot-control-manager
    cmd/iot-consumer
    cmd/iot-data-rollup
    cmd/iotadm
  )
  echo "${targets[@]}"
}

IFS=" " read -ra IOT_SERVER_TARGETS <<< "$(iot::golang::sever_targets)"
readonly IOT_SERVER_TARGETS
readonly IOT_SERVER_BINARIES=("${IOT_SERVER_TARGETS[@]##*/}")

iot::golang::test_targets() {
  local targets=(
    github.com/onsi/ginkgo/ginkgo
    test/e2e/e2e.test
  )

  echo "${targets[@]}"
}

IFS=" " read -ra IOT_TEST_TARGETS <<< "$(iot::golang::test_targets)"
readonly IOT_TEST_TARGETS
readonly IOT_TEST_BINARIES=("${IOT_TEST_TARGETS[@]##*/}")

readonly IOT_ALL_TARGETS=(
  "${IOT_SERVER_TARGETS[@]}"
  "${IOT_TEST_TARGETS[@]}"
)
readonly IOT_ALL_BINARIES=("${IOT_ALL_TARGETS[@]##*/}")

iot::golang::is_statically_linked_library() {
  local e
  [[ "$(go env GOHOSTOS)" == "darwin" && "$(go env GOOS)" == "darwin" &&
    "$1" == *"/kubectl" ]] && return 1
  for e in "${IOT_SERVER_BINARIES[@]}"; do [[ "${1}" == *"/${e}" ]] && return 0; done;
  return 1;
}

# Takes the platform name ($1) and sets the appropriate golang env variables
# for that platform.
iot::golang::set_platform_envs() {
  [[ -n ${1-} ]] || {
    echo "!!! Internal error. No platform set in iot::golang::set_platform_envs"
  }

  export GOOS=${platform%/*}
  export GOARCH=${platform##*/}

  # Do not set CC when building natively on a platform, only if cross-compiling
  if [[ $(iot::golang::host_platform) != "$platform" ]]; then
    # Dynamic CGO linking for other server architectures than host architecture goes here
    # If you want to include support for more server platforms than these, add arch-specific gcc names here
    case "${platform}" in
      "linux/amd64")
        export CGO_ENABLED=1
        export CC=${IOT_LINUX_AMD64_CC:-x86_64-linux-gnu-gcc}
        ;;
      "linux/arm")
        export CGO_ENABLED=1
        export CC=${IOT_LINUX_ARM_CC:-arm-linux-gnueabihf-gcc}
        ;;
      "linux/arm64")
        export CGO_ENABLED=1
        export CC=${IOT_LINUX_ARM64_CC:-aarch64-linux-gnu-gcc}
        ;;
      "linux/ppc64le")
        export CGO_ENABLED=1
        export CC=${IOT_LINUX_PPC64LE_CC:-powerpc64le-linux-gnu-gcc}
        ;;
      "linux/s390x")
        export CGO_ENABLED=1
        export CC=$IOT_LINUX_S390X_CC:-s390x-linux-gnu-gcc}
        ;;
    esac
  fi

  # if CC is defined for platform then always enable it
  ccenv=$(echo "$platform" | awk -F/ '{print "IOT_" toupper($1) "_" toupper($2) "_CC"}')
  if [ -n "${!ccenv-}" ]; then
    export CGO_ENABLED=1
    export CC="${!ccenv}"
  fi
}

# Create the GOPATH tree under $IOT_OUTPUT
iot::golang::create_gopath_tree() {
  local go_pkg_dir="${IOT_GOPATH}/src/${IOT_GO_PACKAGE}"
  local go_pkg_basedir
  go_pkg_basedir=$(dirname "${go_pkg_dir}")

  mkdir -p "${go_pkg_basedir}"

  # TODO: This symlink should be relative.
  if [[ ! -e "${go_pkg_dir}" || "$(readlink "${go_pkg_dir}")" != "${IOT_ROOT}" ]]; then
    ln -snf "${IOT_ROOT}" "${go_pkg_dir}"
  fi
}

# iot::golang::setup_env will check that the `go` commands is available in
# ${PATH}. It will also check that the Go version is good enough for the
# lightiot build.
#
# Inputs:
#   IOT_EXTRA_GOPATH - If set, this is included in created GOPATH
#
# Outputs:
#   env-var GOPATH points to our local output dir
#   env-var GOBIN is unset (we want binaries in a predictable place)
#   env-var GO15VENDOREXPERIMENT=1
#   current directory is within GOPATH
iot::golang::setup_env() {
  iot::golang::create_gopath_tree

  export GOPATH="${IOT_GOPATH}"
  export GOCACHE="${IOT_GOPATH}/cache"

  # Make sure our own Go binaries are in PATH.
  export PATH="${IOT_GOPATH}/bin:${PATH}"

  # Change directories so that we are within the GOPATH.  Some tools get really
  # upset if this is not true.  We use a whole fake GOPATH here to collect the
  # resultant binaries.  Go will not let us use GOBIN with `go install` and
  # cross-compiling, and `go install -o <file>` only works for a single pkg.
  local subdir
  subdir=$(iot::realpath . | sed "s|${IOT_ROOT}||")
  cd "${IOT_GOPATH}/src/${IOT_GO_PACKAGE}/${subdir}" || return 1

  # Set GOROOT so binaries that parse code can work properly.
  GOROOT=$(go env GOROOT)
  export GOROOT

  # Unset GOBIN in case it already exists in the current session.
  unset GOBIN

  # This seems to matter to some tools
  export GO15VENDOREXPERIMENT=1
}

# This will take binaries from $GOPATH/bin and copy them to the appropriate
# place in ${IOT_OUTPUT_BINDIR}
#
# Ideally this wouldn't be necessary and we could just set GOBIN to
# IOT_OUTPUT_BINDIR but that won't work in the face of cross compilation.  'go
# install' will place binaries that match the host platform directly in $GOBIN
# while placing cross compiled binaries into `platform_arch` subdirs.  This
# complicates pretty much everything else we do around packaging and such.
iot::golang::place_bins() {
  local host_platform
  host_platform=$(iot::golang::host_platform)

  echo "Placing binaries"

  local -a platforms
  IFS=" " read -ra platforms <<< "${IOT_BUILD_PLATFORMS:-}"
  if [[ ${#platforms[@]} -eq 0 ]]; then
      platforms=("${host_platform}")
  fi

  local platform
  for platform in "${platforms[@]}"; do
    # The substitution on platform_src below will replace all slashes with
    # underscores.  It'll transform darwin/amd64 -> darwin_amd64.
    local platform_src="/${platform//\//_}"
    if [[ "${platform}" == "${host_platform}" ]]; then
      platform_src=""
      rm -f "${THIS_PLATFORM_BIN}"
      ln -s "${IOT_OUTPUT_BINPATH}/${platform}" "${THIS_PLATFORM_BIN}"
    fi

    local full_binpath_src="${IOT_GOPATH}/bin${platform_src}"
    if [[ -d "${full_binpath_src}" ]]; then
      mkdir -p "${IOT_OUTPUT_BINPATH}/${platform}"
      find "${full_binpath_src}" -maxdepth 1 -type f -exec \
        rsync -pc {} "${IOT_OUTPUT_BINPATH}/${platform}" \;
    fi
  done
}

# Try and replicate the native binary placement of go install without
# calling go install.
iot::golang::outfile_for_binary() {
  local binary=$1
  local platform=$2
  local output_path="${IOT_GOPATH}/bin"
  local bin
  bin=$(basename "${binary}")
  if [[ "${platform}" != "${host_platform}" ]]; then
    output_path="${output_path}/${platform//\//_}"
  fi
  if [[ ${GOOS} == "windows" ]]; then
    bin="${bin}.exe"
  fi
  echo "${output_path}/${bin}"
}

iot::golang::build_binaries_for_platform() {
  # This is for sanity.  Without it, user umasks can leak through.
  umask 0022

  local platform=$1

  local -a statics=()
  local -a nonstatics=()
  local -a tests=()

  echo "Env for ${platform}: GOOS=${GOOS-} GOARCH=${GOARCH-} GOROOT=${GOROOT-} CGO_ENABLED=${CGO_ENABLED-} CC=${CC-}"

  for binary in "${binaries[@]}"; do
    if [[ "${binary}" =~ ".test"$ ]]; then
      tests+=("${binary}")
    elif iot::golang::is_statically_linked_library "${binary}"; then
      statics+=("${binary}")
    else
      nonstatics+=("${binary}")
    fi
  done

  local -a build_args
  if [[ "${#statics[@]}" != 0 ]]; then
    build_args=(
      -installsuffix static
      ${goflags:+"${goflags[@]}"}
      -gcflags "${gogcflags:-}"
      -ldflags "${goldflags:-}"
    )
    echo "> static build CGO_ENABLED=0: ${statics[*]}"
    CGO_ENABLED=0 go install "${build_args[@]}" "${statics[@]}"
  fi

  if [[ "${#nonstatics[@]}" != 0 ]]; then
    build_args=(
      ${goflags:+"${goflags[@]}"}
      -gcflags "${gogcflags:-}"
      -ldflags "${goldflags:-}"
    )
    echo "> non-static build: ${nonstatics[*]}"
    go install "${build_args[@]}" "${nonstatics[@]}"
  fi

  for test in "${tests[@]:+${tests[@]}}"; do
    local outfile testpkg
    outfile=$(iot::golang::outfile_for_binary "${test}" "${platform}")
    testpkg=$(dirname "${test}")

    mkdir -p "$(dirname "${outfile}")"
    go test -c \
      ${goflags:+"${goflags[@]}"} \
      -gcflags "${gogcflags:-}" \
      -ldflags "${goldflags:-}" \
      -o "${outfile}" \
      "${testpkg}"
  done
}

iot::golang::build_binaries() {
    # Create a sub-shell so that we don't pollute the outer environment
    (
        iot::check::env
        iot::golang::setup_env

        local host_platform
        host_platform=$(iot::golang::host_platform)

        local goflags goldflags goasmflags gogcflags gotags
        # If GOLDFLAGS is unset, then set it to the a default of "-s -w".
        # Disable SC2153 for this, as it will throw a warning that the local
        # variable goldflags will exist, and it suggest changing it to this.
        # shellcheck disable=SC2153
        goldflags="${GOLDFLAGS=-s -w -buildid=} $(iot::version::ldflags)"
        gogcflags="${GOGCFLAGS:-} -trimpath=${IOT_ROOT}"

        local -a targets=()
        local arg

        for arg in "$@"; do
            targets+=("${arg}")
        done

        if [[ ${#targets[@]} -eq 0 ]]; then
            targets=("${IOT_ALL_TARGETS[@]}")
        fi

        local -a platforms
        IFS=" " read -ra platforms <<< "${IOT_BUILD_PLATFORMS:-}"
        if [[ ${#platforms[@]} -eq 0 ]]; then
            platforms=("${host_platform}")
        fi

        local -a binaries
        while IFS="" read -r binary; do binaries+=("$binary"); done < <(iot::golang::binaries_from_targets "${targets[@]}")

        for platform in "${platforms[@]}"; do
            echo "Building go targets for ${platform}:" "${targets[@]}"
            (
                iot::golang::set_platform_envs "${platform}"
                iot::golang::build_binaries_for_platform "${platform}"
            )
        done
    )
}
