# Google Calendar API Setup Guide

## Step 1: Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Name your project (e.g., "GTD Study Assistant")
4. Click "Create"

## Step 2: Enable Calendar API

1. In the navigation menu, go to "APIs & Services" → "Library"
2. Search for "Google Calendar API"
3. Click on it and press "Enable"

## Step 3: Create OAuth Credentials

1. Go to "APIs & Services" → "Credentials"
2. Click "Create Credentials" → "OAuth client ID"
3. If prompted, configure the OAuth consent screen:
   - User Type: External (or Internal if using Google Workspace)
   - Fill in App name, user support email, and developer email
   - Add scopes: `https://www.googleapis.com/auth/calendar`
   - Add your email as a test user
4. For Application type, select "Desktop app"
5. Name it "GTD CLI"
6. Click "Create"

## Step 4: Download Credentials

1. Click the download icon next to your newly created OAuth 2.0 Client ID
2. Save the file as `credentials.json`
3. Move it to `~/.credentials/credentials.json`:

```bash
mkdir -p ~/.credentials
mv ~/Downloads/credentials.json ~/.credentials/credentials.json
chmod 600 ~/.credentials/credentials.json
```

## Step 5: First Authorization

Run the application:

```bash
gtd dashboard
```

You'll see a URL in your terminal. Follow these steps:

1. Copy the URL and paste it in your browser
2. Sign in with your Google account
3. Review the permissions and click "Allow"
4. Copy the authorization code
5. Paste it back into the terminal

The token will be saved to `~/.config/gtd/tokens.json` for future use.

## Security Notes

- **Never commit credentials.json or tokens.json to version control**
- Keep file permissions restrictive (600)
- Store credentials in `~/.credentials/` (not in project directory)
- Tokens expire - you may need to re-authorize occasionally

## Troubleshooting

### "Access blocked: This app's request is invalid"

Your OAuth consent screen needs to be properly configured. Go back to step 3 and ensure:
- App name is filled in
- Support email is valid
- Scopes include Calendar API
- Your email is added as a test user

### "The caller does not have permission"

Ensure the Calendar API is enabled (Step 2).

### "invalid_grant" error

Your token has expired. Delete `~/.config/gtd/tokens.json` and re-authorize.

## Rate Limits

Free tier limits:
- 1,000,000 requests per day
- 100 requests per 100 seconds per user

GTD syncs every 5 minutes by default, well within these limits.
