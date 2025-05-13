#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

IOT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${IOT_ROOT}/hack/lib/init.sh"

iot::test::find_dirs() {
  (
    cd "${IOT_ROOT}"
    find -L . -not \( \
        \( \
          -path './apis/*' \
          -o -path './CHANGELOG/*' \
          -o -path './_output/*' \
          -o -path './test/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | sed "s|^\./|${IOT_GO_PACKAGE}/|" | LC_ALL=C sort -u
  )
}

# TODO: This timeout should really be lower, this is a *long* time to test one
# package, however pkg/api/testing in particular will fail with a lower timeout
# currently. We should attempt to lower this over time.
IOT_TIMEOUT=${IOT_TIMEOUT:--timeout=180s}
IOT_COVER=${IOT_COVER:-n} # set to 'y' to enable coverage collection
IOT_COVERMODE=${IOT_COVERMODE:-atomic}
# The directory to save test coverage reports to, if generating them. If unset,
# a semi-predictable temporary directory will be used.
IOT_COVER_REPORT_DIR="${IOT_COVER_REPORT_DIR:-}"
# How many 'go test' instances to run simultaneously when running tests in
# coverage mode.
IOT_COVERPROCS=${IOT_COVERPROCS:-4}
# use IOT_RACE="" to disable the race detector
# this is defaulted to "-race" in make test as well
# NOTE: DO NOT ADD A COLON HERE. IOT_RACE="" is meaningful!
IOT_RACE=${IOT_RACE-"-race"}
# Set to the goveralls binary path to report coverage results to Coveralls.io.
IOT_GOVERALLS_BIN=${IOT_GOVERALLS_BIN:-}
# once we have multiple group supports
# Create a junit-style XML test report in this directory if set.
IOT_JUNIT_REPORT_DIR=${IOT_JUNIT_REPORT_DIR:-}
# If IOT_JUNIT_REPORT_DIR is unset, and ARTIFACTS is set, then have them match.
if [[ -z "${IOT_JUNIT_REPORT_DIR:-}" && -n "${ARTIFACTS:-}" ]]; then
    export IOT_JUNIT_REPORT_DIR="${ARTIFACTS}"
fi
# Set to 'y' to keep the verbose stdout from tests when IOT_JUNIT_REPORT_DIR is
# set.
IOT_KEEP_VERBOSE_TEST_OUTPUT=${IOT_KEEP_VERBOSE_TEST_OUTPUT:-n}

iot::test::usage() {
  cat <<EOF
usage: $0 [OPTIONS] [TARGETS]

OPTIONS:
  -p <number>   : number of parallel workers, must be >= 1
EOF
}

isnum() {
  [[ "$1" =~ ^[0-9]+$ ]]
}

PARALLEL="${PARALLEL:-1}"
while getopts "hp:i:" opt ; do
  case ${opt} in
    h)
      iot::test::usage
      exit 0
      ;;
    p)
      PARALLEL="${OPTARG}"
      if ! isnum "${PARALLEL}" || [[ "${PARALLEL}" -le 0 ]]; then
        echo "'$0': argument to -p must be numeric and greater than 0"
        iot::test::usage
        exit 1
      fi
      ;;
    i)
      echo "'$0': use GOFLAGS='-count <num-iterations>'"
      iot::test::usage
      exit 1
      ;;
    :)
      echo "Option -${OPTARG} <value>"
      iot::test::usage
      exit 1
      ;;
    ?)
      iot::test::usage
      exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

# Use eval to preserve embedded quoted strings.
testargs=()
eval "testargs=(${KUBE_TEST_ARGS:-})"

# Used to filter verbose test output.
go_test_grep_pattern=".*"

# The junit report tool needs full test case information to produce a
# meaningful report.
if [[ -n "${IOT_JUNIT_REPORT_DIR}" ]] ; then
  goflags+=(-v)
  goflags+=(-json)
  # Show only summary lines by matching lines like "status package/test"
  go_test_grep_pattern="^[^[:space:]]\+[[:space:]]\+[^[:space:]]\+/[^[[:space:]]\+"
fi

if [[ -n "${FULL_LOG:-}" ]] ; then
  go_test_grep_pattern=".*"
fi

# Filter out arguments that start with "-" and move them to goflags.
testcases=()
for arg; do
  if [[ "${arg}" == -* ]]; then
    goflags+=("${arg}")
  else
    testcases+=("${arg}")
  fi
done
if [[ ${#testcases[@]} -eq 0 ]]; then
  while IFS='' read -r line; do testcases+=("$line"); done < <(iot::test::find_dirs)
fi
set -- "${testcases[@]+${testcases[@]}}"

if [[ -n "${IOT_RACE}" ]] ; then
  goflags+=("${IOT_RACE}")
fi

junitFilenamePrefix() {
  if [[ -z "${IOT_JUNIT_REPORT_DIR}" ]]; then
    echo ""
    return
  fi
  mkdir -p "${IOT_JUNIT_REPORT_DIR}"
  echo "${IOT_JUNIT_REPORT_DIR}/junit_$(iot::util::sortable_date)"
}

verifyAndSuggestPackagePath() {
  local specified_package_path="$1"
  local alternative_package_path="$2"
  local original_package_path="$3"
  local suggestion_package_path="$4"

  if [[ "${specified_package_path}" =~ '/...'$ ]]; then
    specified_package_path=${specified_package_path::-4}
  fi

  if ! [ -d "${specified_package_path}" ]; then
    # Because k8s sets a localized $GOPATH for testing, seeing the actual
    # directory can be confusing. Instead, just show $GOPATH if it exists in the
    # $specified_package_path.
    local printable_package_path
    printable_package_path=${specified_package_path//${GOPATH}/\$\{GOPATH\}}
    echo "specified test path '${printable_package_path}' does not exist"

    if [ -d "${alternative_package_path}" ]; then
      echo "try changing \"${original_package_path}\" to \"${suggestion_package_path}\""
    fi
    exit 1
  fi
}

verifyPathsToPackagesUnderTest() {
  local packages_under_test=("$@")

  for package_path in "${packages_under_test[@]}"; do
    local local_package_path="${package_path}"
    local go_package_path="${GOPATH}/src/${package_path}"

    if [[ "${package_path:0:2}" == "./" ]] ; then
      verifyAndSuggestPackagePath "${local_package_path}" "${go_package_path}" "${package_path}" "${package_path:2}"
    else
      verifyAndSuggestPackagePath "${go_package_path}" "${local_package_path}" "${package_path}" "./${package_path}"
    fi
  done
}

produceJUnitXMLReport() {
  local -r junit_filename_prefix=$1
  if [[ -z "${junit_filename_prefix}" ]]; then
    return
  fi

  local junit_xml_filename
  junit_xml_filename="${junit_filename_prefix}.xml"

  if ! command -v gotestsum >/dev/null 2>&1; then
    kube::log::status "gotestsum not found; installing from hack/tools"
    pushd "${KUBE_ROOT}/hack/tools" >/dev/null
      GO111MODULE=on go install gotest.tools/gotestsum
    popd >/dev/null
  fi
  gotestsum --junitfile "${junit_xml_filename}" --raw-command cat "${junit_filename_prefix}"*.stdout
  if [[ ! ${IOT_KEEP_VERBOSE_TEST_OUTPUT} =~ ^[yY]$ ]]; then
    rm "${junit_filename_prefix}"*.stdout
  fi

  kube::log::status "Saved JUnit XML test report to ${junit_xml_filename}"
}

runTests() {
  local junit_filename_prefix
  junit_filename_prefix=$(junitFilenamePrefix)

  # verifyPathsToPackagesUnderTest "$@"

  # If we're not collecting coverage, run all requested tests with one 'go test'
  # command, which is much faster.
  if [[ ! ${IOT_COVER} =~ ^[yY]$ ]]; then
    echo "Running tests without code coverage ${IOT_RACE:+"and with ${IOT_RACE}"}"
    go test "${goflags[@]:+${goflags[@]}}" \
     "${IOT_TIMEOUT}" "${@}" \
     "${testargs[@]:+${testargs[@]}}" \
     | tee ${junit_filename_prefix:+"${junit_filename_prefix}.stdout"} \
     | grep --binary-files=text "${go_test_grep_pattern}" && rc=$? || rc=$?
    produceJUnitXMLReport "${junit_filename_prefix}"
    return ${rc}
  fi

  echo "Running tests with code coverage ${IOT_RACE:+"and with ${IOT_RACE}"}"

  # Create coverage report directories.
  if [[ -z "${IOT_COVER_REPORT_DIR}" ]]; then
    cover_report_dir="/tmp/k8s_coverage/$(iot::util::sortable_date)"
  else
    cover_report_dir="${IOT_COVER_REPORT_DIR}"
  fi
  cover_profile="coverage.out"  # Name for each individual coverage profile
  echo "Saving coverage output in '${cover_report_dir}'"
  mkdir -p "${@+${@/#/${cover_report_dir}/}}"

  # Run all specified tests, collecting coverage results. Go currently doesn't
  # support collecting coverage across multiple packages at once, so we must issue
  # separate 'go test' commands for each package and then combine at the end.
  # To speed things up considerably, we can at least use xargs -P to run multiple
  # 'go test' commands at once.
  # To properly parse the test results if generating a JUnit test report, we
  # must make sure the output from PARALLEL runs is not mixed. To achieve this,
  # we spawn a subshell for each PARALLEL process, redirecting the output to
  # separate files.

  # ignore paths:
  # vendor/k8s.io/code-generator/cmd/generator: is fragile when run under coverage, so ignore it for now.
  #                            https://github.com/kubernetes/kubernetes/issues/24967
  # vendor/k8s.io/client-go/1.4/rest: causes cover internal errors
  #                            https://github.com/golang/go/issues/16540
  cover_ignore_dirs="vendor/k8s.io/code-generator/cmd/generator|vendor/k8s.io/client-go/1.4/rest"
  for path in ${cover_ignore_dirs//|/ }; do
      echo -e "skipped\tk8s.io/kubernetes/${path}"
  done

  printf "%s\n" "${@}" \
    | grep -Ev ${cover_ignore_dirs} \
    | xargs -I{} -n 1 -P "${IOT_COVERPROCS}" \
    bash -c "set -o pipefail; _pkg=\"\$0\"; _pkg_out=\${_pkg//\//_}; \
      go test ${goflags[*]:+${goflags[*]}} \
        ${IOT_TIMEOUT} \
        -cover -covermode=\"${IOT_COVERMODE}\" \
        -coverprofile=\"${cover_report_dir}/\${_pkg}/${cover_profile}\" \
        \"\${_pkg}\" \
        ${testargs[*]:+${testargs[*]}} \
      | tee ${junit_filename_prefix:+\"${junit_filename_prefix}-\$_pkg_out.stdout\"} \
      | grep \"${go_test_grep_pattern}\"" \
    {} \
    && test_result=$? || test_result=$?

  produceJUnitXMLReport "${junit_filename_prefix}"

  COMBINED_COVER_PROFILE="${cover_report_dir}/combined-coverage.out"
  {
    # The combined coverage profile needs to start with a line indicating which
    # coverage mode was used (set, count, or atomic). This line is included in
    # each of the coverage profiles generated when running 'go test -cover', but
    # we strip these lines out when combining so that there's only one.
    echo "mode: ${IOT_COVERMODE}"

    # Include all coverage reach data in the combined profile, but exclude the
    # 'mode' lines, as there should be only one.
    while IFS='' read -r x; do
      grep -h -v "^mode:" < "${x}" || true
    done < <(find "${cover_report_dir}" -name "${cover_profile}")
  } >"${COMBINED_COVER_PROFILE}"

  coverage_html_file="${cover_report_dir}/combined-coverage.html"
  go tool cover -html="${COMBINED_COVER_PROFILE}" -o="${coverage_html_file}"
  echo "Combined coverage report: ${coverage_html_file}"

  return ${test_result}
}

reportCoverageToCoveralls() {
  if [[ ${IOT_COVER} =~ ^[yY]$ ]] && [[ -x "${IOT_GOVERALLS_BIN}" ]]; then
    kube::log::status "Reporting coverage results to Coveralls for service ${CI_NAME:-}"
    ${IOT_GOVERALLS_BIN} -coverprofile="${COMBINED_COVER_PROFILE}" \
    ${CI_NAME:+"-service=${CI_NAME}"} \
    ${COVERALLS_REPO_TOKEN:+"-repotoken=${COVERALLS_REPO_TOKEN}"} \
      || true
  fi
}

checkFDs() {
  # several unittests panic when httptest cannot open more sockets
  # due to the low default files limit on OS X.  Warn about low limit.
  local fileslimit
  fileslimit="$(ulimit -n)"
  if [[ ${fileslimit} -lt 1000 ]]; then
    echo "WARNING: ulimit -n (files) should be at least 1000, is ${fileslimit}, may cause test failure";
  fi
}

checkFDs

runTests "$@"

# We might run the tests for multiple versions, but we want to report only
# one of them to coveralls. Here we report coverage from the last run.
reportCoverageToCoveralls
