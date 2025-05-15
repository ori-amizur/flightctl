#!/bin/bash
set -eo pipefail # Exit immediately if a command exits with a non-zero status or a command in a pipeline fails

CONFIG_FILE="/etc/flightctl/service-config.yaml"
TEMPLATES_DIR="/opt/flightctl-observability/templates"
DEFINITIONS_FILE="/etc/flightctl/definitions/otel-collector.defs"

# Source shared logic
source /etc/flightctl/scripts/render-templates.sh

# Call rendering with otel-collector specific definitions
render_templates "$CONFIG_FILE" "$TEMPLATES_DIR" "$DEFINITIONS_FILE" 