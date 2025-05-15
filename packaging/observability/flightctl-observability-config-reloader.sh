#!/bin/bash
set -eo pipefail # Exit immediately if a command exits with a non-zero status or a command in a pipeline fails

CONFIG_FILE="/etc/flightctl/observability_config.yaml"
TEMPLATES_DIR="/opt/flightctl-observability/templates"
GRAFANA_INI_TEMPLATE="${TEMPLATES_DIR}/grafana.ini.template"
GRAFANA_QUADLET_TEMPLATE="${TEMPLATES_DIR}/flightctl-grafana.container.template"

GRAFANA_INI_FINAL="/etc/grafana/grafana.ini"
GRAFANA_QUADLET_FINAL="/etc/containers/systemd/flightctl-grafana.container"

LOG_TAG="flightctl-config-reloader"

# Function to log messages to journal and console
log_message() {
    local level="$1"
    local message="$2"
    logger -t "$LOG_TAG" "$level: $message"
    echo "$level: $message" >&2 # Output to stderr for console visibility
}

log_info() { log_message "INFO" "$1"; }
log_warning() { log_message "WARNING" "$1"; }
log_error() { log_message "ERROR" "$1"; }


log_info "Config file '$CONFIG_FILE' detected change. Re-processing templates..."

# --- Ensure required tools are available ---
if ! command -v /usr/bin/yq &> /dev/null; then
    log_error "'yq' not found. Cannot re-process config. Please install yq."
    exit 1
fi
if ! command -v /usr/bin/envsubst &> /dev/null; then
    log_error "'envsubst' not found. Cannot re-process config. Please install gettext."
    exit 1
fi

# --- Export variables from YAML for envsubst ---
# Use "|| echo 'default_value'" for safe defaults if a path is missing or null.
export OBSERVABILITY_GRAFANA_OAUTH_ENABLED=$(/usr/bin/yq e '.grafana.oauth.enabled' "$CONFIG_FILE" 2>/dev/null || echo "false")
export OBSERVABILITY_GRAFANA_PUBLISHED_PORT=$(/usr/bin/yq e '.grafana.published_port' "$CONFIG_FILE" 2>/dev/null || echo "3000")
export OBSERVABILITY_GRAFANA_OAUTH_CLIENT_ID=$(/usr/bin/yq e '.grafana.oauth.client_id' "$CONFIG_FILE" 2>/dev/null || echo "")
# No client_secret needed for PKCE-enabled Public Clients
export OBSERVABILITY_GRAFANA_LOCAL_ADMIN_USER=$(/usr/bin/yq e '.grafana.oauth.local_admin_user' "$CONFIG_FILE" 2>/dev/null || echo "admin")
export OBSERVABILITY_GRAFANA_LOCAL_ADMIN_PASSWORD=$(/usr/bin/yq e '.grafana.oauth.local_admin_password' "$CONFIG_FILE" 2>/dev/null || echo "defaultadmin")

# Calculate derived OAuth URLs from provider_base_url and realm_name
OAUTH_PROVIDER_BASE_URL=$(/usr/bin/yq e '.grafana.oauth.provider_base_url' "$CONFIG_FILE" 2>/dev/null)
OAUTH_REALM_NAME=$(/usr/bin/yq e '.grafana.oauth.realm_name' "$CONFIG_FILE" 2>/dev/null)

if [ -n "$OAUTH_PROVIDER_BASE_URL" ] && [ "$OAUTH_PROVIDER_BASE_URL" != "null" ] && \
   [ -n "$OAUTH_REALM_NAME" ] && [ "$OAUTH_REALM_NAME" != "null" ]; then
    export OBSERVABILITY_GRAFANA_OAUTH_AUTH_URL="$OAUTH_PROVIDER_BASE_URL/realms/$OAUTH_REALM_NAME/protocol/openid-connect/auth"
    export OBSERVABILITY_GRAFANA_OAUTH_TOKEN_URL="$OAUTH_PROVIDER_BASE_URL/realms/$OAUTH_REALM_NAME/protocol/openid-connect/token"
    export OBSERVABILITY_GRAFANA_OAUTH_API_URL="$OAUTH_PROVIDER_BASE_URL/realms/$OAUTH_REALM_NAME/protocol/openid-connect/userinfo"
    export OBSERVABILITY_GRAFANA_OAUTH_JWKS_URL="$OAUTH_PROVIDER_BASE_URL/realms/$OAUTH_REALM_NAME/protocol/openid-connect/certs"
else
    log_warning "OAuth provider_base_url or realm_name not found/null. OAuth URLs not set. OAuth will be disabled."
    export OBSERVABILITY_GRAFANA_OAUTH_ENABLED="false" # Force disable OAuth if URLs are incomplete
    export OBSERVABILITY_GRAFANA_OAUTH_AUTH_URL="" # Clear derived URLs if not valid
    export OBSERVABILITY_GRAFANA_OAUTH_TOKEN_URL=""
    export OBSERVABILITY_GRAFANA_OAUTH_API_URL=""
    export OBSERVABILITY_GRAFANA_OAUTH_JWKS_URL=""
fi

# --- Process templates with envsubst ---
# The list of variables needs to be explicitly passed to envsubst
log_info "Generating $GRAFANA_INI_FINAL from template..."
/usr/bin/envsubst \
    '${OBSERVABILITY_GRAFANA_OAUTH_ENABLED}:${OBSERVABILITY_GRAFANA_OAUTH_CLIENT_ID}:${OBSERVABILITY_GRAFANA_OAUTH_AUTH_URL}:${OBSERVABILITY_GRAFANA_OAUTH_TOKEN_URL}:${OBSERVABILITY_GRAFANA_OAUTH_API_URL}:${OBSERVABILITY_GRAFANA_OAUTH_JWKS_URL}:${OBSERVABILITY_GRAFANA_LOCAL_ADMIN_USER}:${OBSERVABILITY_GRAFANA_LOCAL_ADMIN_PASSWORD}' \
    < "$GRAFANA_INI_TEMPLATE" > "$GRAFANA_INI_FINAL" || { log_error "Failed to generate $GRAFANA_INI_FINAL"; exit 1; }

log_info "Generating $GRAFANA_QUADLET_FINAL from template..."
/usr/bin/envsubst \
    '${OBSERVABILITY_GRAFANA_PUBLISHED_PORT}:${OBSERVABILITY_GRAFANA_OAUTH_ENABLED}' \
    < "$GRAFANA_QUADLET_TEMPLATE" > "$GRAFANA_QUADLET_FINAL" || { log_error "Failed to generate $GRAFANA_QUADLET_FINAL"; exit 1; }

# --- Apply permissions and SELinux context to generated files ---
log_info "Applying permissions and SELinux context to generated files..."
chmod 0644 "$GRAFANA_INI_FINAL" || { log_error "Failed to chmod $GRAFANA_INI_FINAL"; exit 1; }
chmod 0644 "$GRAFANA_QUADLET_FINAL" || { log_error "Failed to chmod $GRAFANA_QUADLET_FINAL"; exit 1; }
/usr/sbin/restorecon "$GRAFANA_INI_FINAL" || { log_error "Failed to restorecon $GRAFANA_INI_FINAL"; exit 1; }
/usr/sbin/restorecon "$GRAFANA_QUADLET_FINAL" || { log_error "Failed to restorecon $GRAFANA_QUADLET_FINAL"; exit 1; }
# Ensure existing non-templated file also has correct context
restorecon "/etc/grafana/provisioning/datasources/prometheus.yaml" || { log_error "Failed to restorecon /etc/grafana/provisioning/datasources/prometheus.yaml"; exit 1; }

# --- Unset exported variables to clean up the shell environment ---
unset OBSERVABILITY_GRAFANA_OAUTH_ENABLED \
      OBSERVABILITY_GRAFANA_PUBLISHED_PORT \
      OBSERVABILITY_GRAFANA_OAUTH_CLIENT_ID \
      OBSERVABILITY_GRAFANA_OAUTH_AUTH_URL \
      OBSERVABILITY_GRAFANA_OAUTH_TOKEN_URL \
      OBSERVABILITY_GRAFANA_OAUTH_API_URL \
      OBSERVABILITY_GRAFANA_OAUTH_JWKS_URL \
      OBSERVABILITY_GRAFANA_LOCAL_ADMIN_USER \
      OBSERVABILITY_GRAFANA_LOCAL_ADMIN_PASSWORD \
      OAUTH_PROVIDER_BASE_URL OAUTH_REALM_NAME

# --- Reload systemd and restart Grafana ---
log_info "Templates processed. Reloading systemd and restarting Grafana..."
/usr/bin/systemctl daemon-reload || { log_error "Failed to daemon-reload"; exit 1; }
/usr/bin/systemctl restart flightctl-grafana.service || { log_error "Failed to restart flightctl-grafana.service"; exit 1; }
log_info "Grafana restarted successfully."

exit 0
