#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

IOT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${IOT_ROOT}/hack/lib/init.sh"

# Find the ginkgo binary build as part of the release.
ginkgo=$(iot::util::find-binary "ginkgo")
e2e_test=$(iot::util::find-binary "e2e.test")

# --- Setup some env vars.

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
CLOUD_CONFIG=${CLOUD_CONFIG:-""}

# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}

# If 'y', will rerun failed tests once to give them a second chance.
GINKGO_TOLERATE_FLAKES=${GINKGO_TOLERATE_FLAKES:-n}

# If set, the command executed will be:
# - `dlv exec` if set to "delve"
# - `gdb` if set to "gdb"
# NOTE: for this to work the e2e.test binary has to be compiled with
# make WHAT=test/e2e/e2e.test GOGCFLAGS="all=-N -l" GOLDFLAGS=""
E2E_TEST_DEBUG_TOOL=${E2E_TEST_DEBUG_TOOL:-}

ginkgo_args=()
if [[ -n "${CONFORMANCE_TEST_SKIP_REGEX:-}" ]]; then
  ginkgo_args+=("--skip=${CONFORMANCE_TEST_SKIP_REGEX}")
  ginkgo_args+=("--seed=1436380640")
fi
if [[ -n "${GINKGO_PARALLEL_NODES:-}" ]]; then
  ginkgo_args+=("--nodes=${GINKGO_PARALLEL_NODES}")
elif [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
  ginkgo_args+=("--nodes=25")
fi

if [[ "${GINKGO_UNTIL_IT_FAILS:-}" == true ]]; then
  ginkgo_args+=("--untilItFails=true")
fi

FLAKE_ATTEMPTS=1
if [[ "${GINKGO_TOLERATE_FLAKES}" == "y" ]]; then
  FLAKE_ATTEMPTS=2
fi

if [[ "${GINKGO_NO_COLOR}" == "y" ]]; then
  ginkgo_args+=("--noColor")
fi
if [[ -n "${GINKGO_LABEL_FILTER:-}" ]]; then
  ginkgo_args+=("--label-filter=${GINKGO_LABEL_FILTER}")
fi

CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-${KUBE_CONTAINER_RUNTIME:-}}

# The --host setting is used only when providing --auth_config
# If --kubeconfig is used, the host to use is retrieved from the .kubeconfig
# file and the one provided with --host is ignored.
# Add path for things like running kubectl binary.
PATH=$(dirname "${e2e_test}"):"${PATH}"
export PATH

# Choose the program to execute.
program=("${ginkgo}")
if [[ "${E2E_TEST_DEBUG_TOOL}" == "delve" ]]; then
  program=("dlv" "exec")
elif [[ "${E2E_TEST_DEBUG_TOOL}" == "gdb" ]]; then
  program=("gdb")
fi

"${program[@]}" "${ginkgo_args[@]:+${ginkgo_args[@]}}" "${e2e_test}" -- \
  "${auth_config[@]:+${auth_config[@]}}" \
  --ginkgo.flake-attempts="${FLAKE_ATTEMPTS}" \
  --ginkgo.slow-spec-threshold="${GINKGO_SLOW_SPEC_THRESHOLD:-300s}" \
  ${E2E_REPORT_DIR:+"--report-dir=${E2E_REPORT_DIR}"} \
  ${E2E_REPORT_PREFIX:+"--report-prefix=${E2E_REPORT_PREFIX}"} \
  ${IOT_MODEL_MANAGER_HOST:+"--iot-model-manager-host=${IOT_MODEL_MANAGER_HOST}"} \
  ${IOT_RULE_MANAGER_HOST:+"--iot-rule-manager-host=${IOT_RULE_MANAGER_HOST}"} \
  "${@:-}"
