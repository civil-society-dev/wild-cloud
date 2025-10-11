#!/bin/bash
# Common functions for Wild Central service installation scripts

# TODO: We should use this. :P

# Ensure required environment variables are set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "❌ ERROR: WILD_INSTANCE environment variable is not set"
    exit 1
fi

if [ -z "${WILD_CENTRAL_DATA}" ]; then
    echo "❌ ERROR: WILD_CENTRAL_DATA environment variable is not set"
    exit 1
fi

# Get the instance directory path
get_instance_dir() {
    echo "${WILD_CENTRAL_DATA}/instances/${WILD_INSTANCE}"
}

# Get the secrets file path
get_secrets_file() {
    echo "$(get_instance_dir)/secrets.yaml"
}

# Get the config file path
get_config_file() {
    echo "$(get_instance_dir)/config.yaml"
}

# Get a secret value from the secrets file
# Usage: get_secret "path.to.secret"
get_secret() {
    local path="$1"
    local secrets_file="$(get_secrets_file)"

    if [ ! -f "$secrets_file" ]; then
        echo ""
        return 1
    fi

    local value=$(yq ".$path" "$secrets_file" 2>/dev/null)

    # Remove quotes and return empty string if null
    value=$(echo "$value" | tr -d '"')
    if [ "$value" = "null" ]; then
        echo ""
        return 1
    fi

    echo "$value"
}

# Get a config value from the config file
# Usage: get_config "path.to.config"
get_config() {
    local path="$1"
    local config_file="$(get_config_file)"

    if [ ! -f "$config_file" ]; then
        echo ""
        return 1
    fi

    local value=$(yq ".$path" "$config_file" 2>/dev/null)

    # Remove quotes and return empty string if null
    value=$(echo "$value" | tr -d '"')
    if [ "$value" = "null" ]; then
        echo ""
        return 1
    fi

    echo "$value"
}

# Check if a secret exists and is not empty
# Usage: require_secret "path.to.secret" "Friendly Name" "wild secret set command"
require_secret() {
    local path="$1"
    local name="$2"
    local set_command="$3"

    local value=$(get_secret "$path")

    if [ -z "$value" ]; then
        echo "❌ ERROR: $name not found"
        echo "💡 Please set: $set_command"
        exit 1
    fi

    echo "$value"
}

# Check if a config value exists and is not empty
# Usage: require_config "path.to.config" "Friendly Name" "wild config set command"
require_config() {
    local path="$1"
    local name="$2"
    local set_command="$3"

    local value=$(get_config "$path")

    if [ -z "$value" ]; then
        echo "❌ ERROR: $name not found"
        echo "💡 Please set: $set_command"
        exit 1
    fi

    echo "$value"
}
