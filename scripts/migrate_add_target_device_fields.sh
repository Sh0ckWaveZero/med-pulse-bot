#!/bin/bash

# Migration Script: Add target_device fields to employee_detections collection
# Usage: ./scripts/migrate_add_target_device_fields.sh

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Load .env file
if [ -f "$PROJECT_ROOT/.env" ]; then
  echo "📝 Loading .env file..."
  export $(cat "$PROJECT_ROOT/.env" | grep -v '^#' | xargs)
else
  echo "⚠️  Warning: .env file not found at $PROJECT_ROOT/.env"
fi

# Use environment variables or defaults
POCKETBASE_URL="${POCKETBASE_URL:-http://192.168.100.100:8090}"
TOKEN="${POCKETBASE_TOKEN}"

echo "🔧 PocketBase Migration: Add Target Device Fields"
echo "=================================================="
echo "📍 PocketBase URL: $POCKETBASE_URL"
echo ""

# Function to check if PocketBase is running
check_pocketbase() {
  echo "🔍 Checking PocketBase connection..."
  if ! curl -s -f "$POCKETBASE_URL/api/health" > /dev/null 2>&1; then
    echo "❌ Error: Cannot connect to PocketBase at $POCKETBASE_URL"
    echo "💡 Make sure PocketBase is running: docker-compose up -d"
    exit 1
  fi
  echo "✅ PocketBase is running"
  echo ""
}

# Function to check token
check_token() {
  echo "🔐 Checking authentication token..."

  if [ -z "$TOKEN" ]; then
    echo "❌ Error: POCKETBASE_TOKEN not found in .env file"
    echo "📝 Please add POCKETBASE_TOKEN to your .env file"
    exit 1
  fi

  echo "✅ Token found"
  echo ""
}

# Function to get current collection schema
get_collection() {
  echo "📖 Fetching employee_detections collection..."

  COLLECTION=$(curl -s -X GET "$POCKETBASE_URL/api/collections/employee_detections" \
    -H "Authorization: $TOKEN")

  if echo "$COLLECTION" | grep -q '"id"'; then
    echo "✅ Collection found"
    echo ""
    return 0
  else
    echo "❌ Collection not found"
    echo "Response: $COLLECTION"
    exit 1
  fi
}

# Function to add new fields
migrate() {
  echo "🔄 Adding new fields to employee_detections..."
  echo "   • is_target_device (Boolean)"
  echo "   • device_name (Text, max 255)"
  echo ""

  # Get current fields
  CURRENT_FIELDS=$(echo "$COLLECTION" | jq -r '.fields')

  # Check if fields already exist
  if echo "$CURRENT_FIELDS" | grep -q "is_target_device"; then
    echo "⚠️  Field 'is_target_device' already exists. Skipping..."
    return 0
  fi

  if echo "$CURRENT_FIELDS" | grep -q "device_name"; then
    echo "⚠️  Field 'device_name' already exists. Skipping..."
    return 0
  fi

  # Get collection ID for the update
  COLLECTION_ID=$(echo "$COLLECTION" | jq -r '.id')

  # Add new fields using PocketBase API format
  NEW_FIELDS=$(echo "$COLLECTION" | jq '.fields += [
    {
      "name": "is_target_device",
      "type": "bool",
      "required": false,
      "presentable": false,
      "hidden": false,
      "system": false
    },
    {
      "name": "device_name",
      "type": "text",
      "required": false,
      "presentable": false,
      "hidden": false,
      "system": false,
      "autogeneratePattern": "",
      "pattern": "",
      "min": 0,
      "max": 255,
      "primaryKey": false
    }
  ]')

  # Create update payload
  UPDATED_SCHEMA=$(echo "$NEW_FIELDS" | jq '{
    name: .name,
    type: .type,
    fields: .fields
  }')

  # Update collection
  RESPONSE=$(curl -s -X PATCH "$POCKETBASE_URL/api/collections/$COLLECTION_ID" \
    -H "Authorization: $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$UPDATED_SCHEMA")

  if echo "$RESPONSE" | grep -q '"id"'; then
    echo "✅ Migration completed successfully!"
    echo ""
    echo "📊 Updated Fields:"
    echo "$RESPONSE" | jq -r '.fields[] | select(.name == "is_target_device" or .name == "device_name")'
  else
    echo "❌ Migration failed"
    echo "Response: $RESPONSE"
    exit 1
  fi
}

# Function to verify migration
verify() {
  echo ""
  echo "🧪 Verifying migration..."

  COLLECTION=$(curl -s -X GET "$POCKETBASE_URL/api/collections/employee_detections" \
    -H "Authorization: $TOKEN")

  if echo "$COLLECTION" | jq -r '.fields[].name' | grep -q "is_target_device"; then
    echo "✅ Field 'is_target_device' verified"
  else
    echo "❌ Field 'is_target_device' not found"
    exit 1
  fi

  if echo "$COLLECTION" | jq -r '.fields[].name' | grep -q "device_name"; then
    echo "✅ Field 'device_name' verified"
  else
    echo "❌ Field 'device_name' not found"
    exit 1
  fi

  echo ""
  echo "🎉 Migration verified successfully!"
  echo ""
  echo "📋 Next Steps:"
  echo "   1. Restart Backend API: docker-compose restart app"
  echo "   2. Upload firmware to ESP32"
  echo "   3. Test target device detection"
}

# Main execution
main() {
  check_pocketbase
  check_token
  get_collection
  migrate
  verify
}

main
