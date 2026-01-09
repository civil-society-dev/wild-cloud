#!/bin/bash
set -e
set -o pipefail

# Initialize Wild Cloud environment
if [ -z "${WC_ROOT}" ]; then
    print "WC_ROOT is not set."
    exit 1
else
    source "${WC_ROOT}/scripts/common.sh"
    init_wild_env
fi

print_header "Setting up backup configuration"

print_info "Backup configuration allows Wild Cloud applications to create and manage backups"
print_info "(database backups, file backups, etc.)."
echo ""

# Collect backup configuration
print_info "Collecting backup configuration..."
prompt_if_unset_config "cloud.backup.root" "Enter path for backups" ""
prompt_if_unset_config "cloud.backup.staging" "Enter path for staging backups" ""
print_success "Backup configuration collected successfully"
