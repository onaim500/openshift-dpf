#!/bin/bash

set -e
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/env.sh"
source "${SCRIPT_DIR}/utils.sh"

declare -a failed_steps=()

run_cleanup_step() {
    local description="$1"
    shift

    log "INFO" "${description}..."
    local status=0
    if "$@"; then
        return 0
    else
        status=$?
    fi

    log "ERROR" "${description} failed with exit code ${status}: $*"
    failed_steps+=("${description}")
}

run_cleanup_step "Step 1/4: Gracefully shut down physical workers" "${SCRIPT_DIR}/worker.sh" shutoff-all-workers
run_cleanup_step "Step 2/4: Destroy worker VMs" "${SCRIPT_DIR}/vm.sh" delete-worker-vms
run_cleanup_step "Step 3/4: Delete Assisted Installer cluster and generated files" "${SCRIPT_DIR}/cluster.sh" clean-all
run_cleanup_step "Step 4/4: Destroy remaining VMs" "${SCRIPT_DIR}/vm.sh" delete

if ((${#failed_steps[@]} > 0)); then
    log "ERROR" "Cleanup finished with ${#failed_steps[@]} failed step(s): ${failed_steps[*]}"
    exit 1
fi

log "INFO" "All cleanup steps completed successfully"
