#!/usr/bin/env bash

set -eo pipefail

# Load shared functions first to get the constant directory paths
SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
source "${SCRIPT_DIR}"/shared.sh

echo "Starting Deployment"

# Run installation script
if ! deploy/scripts/install.sh; then
    echo "Error: Installation failed"
    exit 1
fi

echo "Ensuring secrets are available..."
# Always ensure secrets exist before starting services
if ! /usr/share/flightctl/init_host.sh; then
    echo "Error: Failed to initialize secrets"
    exit 1
fi

echo "Starting all FlightCtl services via target..."
start_service "flightctl.target"

echo "Waiting for core services to initialize..."
# Wait for database to be ready first
timeout --foreground 120s bash -c '
    while true; do
        if podman ps --quiet --filter "name=flightctl-db" | grep -q . && \
           podman exec flightctl-db pg_isready -U postgres >/dev/null 2>&1; then
            echo "Database is ready"
            break
        fi
        echo "Waiting for database to become ready..."
        sleep 3
    done
'

# Sync database password with secrets
echo "Synchronizing database password with secrets..."
DB_ACTUAL_PASSWORD=$(sudo podman exec flightctl-db printenv POSTGRESQL_MASTER_PASSWORD)
if ! sudo podman run --rm --network flightctl \
    --secret flightctl-postgresql-master-password,type=env,target=DB_PASSWORD \
    quay.io/sclorg/postgresql-16-c9s:latest \
    bash -c 'PGPASSWORD="$DB_PASSWORD" psql -h flightctl-db -U admin -d flightctl -c "SELECT 1" >/dev/null 2>&1'; then
    
    echo "Password mismatch detected! Fixing secret..."
    sudo podman secret rm flightctl-postgresql-master-password
    echo "$DB_ACTUAL_PASSWORD" | sudo podman secret create flightctl-postgresql-master-password -
    echo "Secret updated to match database password"
fi

# Ensure admin has superuser privileges
echo "Ensuring database admin has superuser privileges..."
if sudo podman exec flightctl-db psql -U postgres -tAc "SELECT rolsuper FROM pg_roles WHERE rolname = 'admin'" | grep -q "f"; then
    echo "Granting superuser privileges to admin user..."
    sudo podman exec flightctl-db psql -U postgres -c "ALTER USER admin WITH SUPERUSER;"
fi

# Wait for key-value service
timeout --foreground 60s bash -c '
    while true; do
        if podman ps --quiet --filter "name=flightctl-kv" | grep -q . && \
           podman exec flightctl-kv redis-cli ping >/dev/null 2>&1; then
            echo "Key-value service is ready"
            break
        fi
        echo "Waiting for key-value service..."
        sleep 2
    done
'

# Restart any failed services due to initial password/permission issues
echo "Restarting API services to apply fixes..."
sudo systemctl restart flightctl-api.service flightctl-worker.service flightctl-periodic.service

echo "Waiting for all services to be fully ready..."
timeout --foreground 120s bash -c '
    while true; do
        if systemctl is-active --quiet flightctl.target && \
           systemctl is-active --quiet flightctl-api.service && \
           systemctl is-active --quiet flightctl-worker.service && \
           systemctl is-active --quiet flightctl-periodic.service; then
            echo "All services are active and ready"
            break
        fi
        echo "Waiting for services to become fully active..."
        sleep 3
    done
'

echo "Deployment completed successfully!"
echo ""
echo "All FlightCtl services should now be running:"
echo "  - Database (PostgreSQL)"
echo "  - Key-Value Store (Redis)" 
echo "  - API Server"
echo "  - Worker Service"
echo "  - Periodic Service"
echo ""
echo "You can check status with: sudo systemctl status flightctl.target"
