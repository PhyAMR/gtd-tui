#!/bin/bash
# Generate Google Calendar API credentials setup script

echo "=== Google Calendar API Credentials Setup ==="
echo
echo "This script will guide you through setting up Google Calendar API credentials."
echo
echo "Prerequisites:"
echo "  1. A Google account"
echo "  2. Access to Google Cloud Console"
echo
read -p "Press Enter to continue..."

echo
echo "Step 1: Create a Google Cloud Project"
echo "  → Go to: https://console.cloud.google.com/"
echo "  → Click 'New Project'"
echo "  → Name it (e.g., 'GTD Study Assistant')"
echo
read -p "Press Enter when done..."

echo
echo "Step 2: Enable Calendar API"
echo "  → Go to: APIs & Services → Library"
echo "  → Search for 'Google Calendar API'"
echo "  → Click 'Enable'"
echo
read -p "Press Enter when done..."

echo
echo "Step 3: Create OAuth Credentials"
echo "  → Go to: APIs & Services → Credentials"
echo "  → Click 'Create Credentials' → 'OAuth client ID'"
echo "  → Configure OAuth consent screen if prompted"
echo "  → Select 'Desktop app' as application type"
echo "  → Download the credentials JSON"
echo
read -p "Press Enter when done..."

echo
echo "Step 4: Install credentials"
read -p "Enter the path to your downloaded credentials.json: " CRED_PATH

if [ ! -f "$CRED_PATH" ]; then
    echo "Error: File not found: $CRED_PATH"
    exit 1
fi

mkdir -p ~/.credentials
cp "$CRED_PATH" ~/.credentials/credentials.json
chmod 600 ~/.credentials/credentials.json

echo
echo "✓ Credentials installed to ~/.credentials/credentials.json"
echo
echo "You're all set! Run 'gtd dashboard' to start."
echo "You'll be prompted to authorize on first run."
