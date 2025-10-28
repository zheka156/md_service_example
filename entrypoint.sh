#!/bin/sh
set -e

echo "Waiting for postgres..."
sleep 5

echo "Running migrations..."
goose -dir /root/migrations postgres "${DB_URL}" up

echo "Starting application..."
exec ./md_service