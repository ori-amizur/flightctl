#!/bin/bash
set -eo pipefail # Exit immediately if a command exits with a non-zero status or a command in a pipeline fails

#!/bin/bash

CONFIG_FILE="/etc/flightctl/service-config.yaml"
TEMPLATES_DIR="/opt/flightctl-observability/templates"
DEFINITIONS_FILE="/etc/flightctl/definitions/observability.defs"

# Source shared logic
source /etc/flightctl/scripts/render-templates.sh

# Call rendering with provided definitions
render_templates "$CONFIG_FILE" "$TEMPLATES_DIR" "$DEFINITIONS_FILE"

