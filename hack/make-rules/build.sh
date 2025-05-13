#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

IOT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${IOT_ROOT}/hack/lib/init.sh"

iot::golang::build_binaries "$@"
iot::golang::place_bins
