#!/bin/bash

# This script sets up a go workspace locally and run the autoheal receiver.
source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

# Get host plataform
platform="$(os::build::host_platform)"

# Get platform binary and extention
binary="${OS_OUTPUT_BINPATH}/${platform}/autoheal"
if [[ $platform == "windows/amd64" ]]; then
	binary=("${binary}.exe")
else
	binary=("${binary}")
fi

# Run autoheal receiver using dev defaults
"${binary}" server --config-file=examples/autoheal-dev.yml --logtostderr
