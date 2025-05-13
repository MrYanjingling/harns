#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Version management helpers.  These functions help to set, save and load the
# following variables:
#
#    IOT_GIT_COMMIT - The git commit id corresponding to this
#          source code.
#    IOT_GIT_TREE_STATE - "clean" indicates no changes since the git commit id
#        "dirty" indicates source code changes after the git commit id
#        "archive" indicates the tree was produced by 'git archive'
#    IOT_GIT_VERSION - "vX.Y" used to indicate the last release version.
#    IOT_GIT_MAJOR - The major part of the version
#    IOT_GIT_MINOR - The minor component of the version

# Grovels through git to set a set of env variables.
#
# If IOT_GIT_VERSION_FILE, this function will load from that file instead of
# querying git.
iot::version::get_version_vars() {
  if [[ -n ${IOT_GIT_VERSION_FILE-} ]]; then
    iot::version::load_version_vars "${IOT_GIT_VERSION_FILE}"
    return
  fi

  local git=(git --work-tree "${IOT_ROOT}")

  if [[ -n ${IOT_GIT_COMMIT-} ]] || IOT_GIT_COMMIT=$("${git[@]}" rev-parse "HEAD^{commit}" 2>/dev/null); then
    if [[ -z ${IOT_GIT_TREE_STATE-} ]]; then
      # Check if the tree is dirty.  default to dirty
      if git_status=$("${git[@]}" status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
        IOT_GIT_TREE_STATE="clean"
      else
        IOT_GIT_TREE_STATE="dirty"
      fi
    fi

    # Use git describe to find the version based on tags.
    if [[ -n ${IOT_GIT_VERSION-} ]] || IOT_GIT_VERSION=$("${git[@]}" describe --tags --match='v*' --abbrev=14 "${IOT_GIT_COMMIT}^{commit}" 2>/dev/null); then
      # This translates the "git describe" to an actual semver.org
      # compatible semantic version that looks something like this:
      #   v1.1.0-alpha.0.6+84c76d1142ea4d
      #
      # TODO: We continue calling this "git version" because so many
      # downstream consumers are expecting it there.
      #
      # These regexes are painful enough in sed...
      # We don't want to do them in pure shell, so disable SC2001
      # shellcheck disable=SC2001
      DASHES_IN_VERSION=$(echo "${IOT_GIT_VERSION}" | sed "s/[^-]//g")
      if [[ "${DASHES_IN_VERSION}" == "---" ]] ; then
        # shellcheck disable=SC2001
        # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
        IOT_GIT_VERSION=$(echo "${IOT_GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\+\2/")
      elif [[ "${DASHES_IN_VERSION}" == "--" ]] ; then
        # shellcheck disable=SC2001
        # We have distance to base tag (v1.1.0-1-gCommitHash)
        IOT_GIT_VERSION=$(echo "${IOT_GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/+\1/")
      fi
      if [[ "${IOT_GIT_TREE_STATE}" == "dirty" ]]; then
        # git describe --dirty only considers changes to existing files, but
        # that is problematic since new untracked .go files affect the build,
        # so use our idea of "dirty" from git status instead.
        IOT_GIT_VERSION+="-dirty"
      fi


      # Try to match the "git describe" output to a regex to try to extract
      # the "major" and "minor" versions and whether this is the exact tagged
      # version or whether the tree is between two tagged versions.
      if [[ "${IOT_GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?([-].*)?([+].*)?$ ]]; then
        IOT_GIT_MAJOR=${BASH_REMATCH[1]}
        IOT_GIT_MINOR=${BASH_REMATCH[2]}
        if [[ -n "${BASH_REMATCH[4]}" ]]; then
          IOT_GIT_MINOR+="+"
        fi
      fi

      # If IOT_GIT_VERSION is not a valid Semantic Version, then refuse to build.
      if ! [[ "${IOT_GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
          echo "IOT_GIT_VERSION should be a valid Semantic Version. Current value: ${IOT_GIT_VERSION}"
          echo "Please see more details here: https://semver.org"
          exit 1
      fi
    fi
  fi
}

# Saves the environment flags to $1
iot::version::save_version_vars() {
  local version_file=${1-}
  [[ -n ${version_file} ]] || {
    echo "!!! Internal error.  No file specified in iot::version::save_version_vars"
    return 1
  }

  cat <<EOF >"${version_file}"
IOT_GIT_COMMIT='${IOT_GIT_COMMIT-}'
IOT_GIT_TREE_STATE='${IOT_GIT_TREE_STATE-}'
IOT_GIT_VERSION='${IOT_GIT_VERSION-}'
IOT_GIT_MAJOR='${IOT_GIT_MAJOR-}'
IOT_GIT_MINOR='${IOT_GIT_MINOR-}'
EOF
}

# Loads up the version variables from file $1
iot::version::load_version_vars() {
  local version_file=${1-}
  [[ -n ${version_file} ]] || {
    echo "!!! Internal error.  No file specified in iot::version::load_version_vars"
    return 1
  }

  source "${version_file}"
}

# Prints the value that needs to be passed to the -ldflags parameter of go build
# in order to set the Kubernetes based on the git tree status.
# IMPORTANT: if you update any of these, also update the lists in
# pkg/version.
iot::version::ldflags() {
  iot::version::get_version_vars

  local -a ldflags
  function add_ldflag() {
    local key=${1}
    local val=${2}
    ldflags+=(
      "-X '${IOT_GO_PACKAGE}/pkg/version.${key}=${val}'"
    )
  }

  add_ldflag "buildDate" "$(date ${SOURCE_DATE_EPOCH:+"--date=@${SOURCE_DATE_EPOCH}"} -u +'%Y-%m-%dT%H:%M:%SZ')"
  if [[ -n ${IOT_GIT_COMMIT-} ]]; then
    add_ldflag "gitCommit" "${IOT_GIT_COMMIT}"
    add_ldflag "gitTreeState" "${IOT_GIT_TREE_STATE}"
  fi

  if [[ -n ${IOT_GIT_VERSION-} ]]; then
    add_ldflag "gitVersion" "${IOT_GIT_VERSION}"
  fi

  if [[ -n ${IOT_GIT_MAJOR-} && -n ${IOT_GIT_MINOR-} ]]; then
    add_ldflag "gitMajor" "${IOT_GIT_MAJOR}"
    add_ldflag "gitMinor" "${IOT_GIT_MINOR}"
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}
