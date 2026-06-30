# LinkedIn OAuth Setup

Prerequisites for connecting velocirepo to the LinkedIn API.

## 1. Create a LinkedIn Developer App

1. Go to https://www.linkedin.com/developers/apps/new
2. Fill in:
   - **App name**: e.g., "velocirepo"
   - **LinkedIn Page**: select your company page (requires Super Admin role)
   - **App logo**: any image (required but not important)
3. Accept the Legal Agreement and click **Create app**

## 2. Request Community Management API Access

1. In your app, go to the **Products** tab
2. Find **Community Management API** and click **Request access**
3. As a Super Admin, approval is typically immediate

This grants the `r_organization_social` scope needed to read posts and engagement stats.

## 3. Configure the Redirect URL

1. Go to the **Auth** tab in your app settings
2. Under **Authorized redirect URLs for your app**, add:
   ```
   http://localhost:9876/callback
   ```
3. Click **Update**

## 4. Note Your Credentials

From the **Auth** tab, copy:

- **Client ID** (visible by default)
- **Primary Client Secret** (click the eye icon to reveal)

You will need these when running `velocirepo auth linkedin`.

## 5. Authenticate

Run the supported OAuth helper:

```bash
velocirepo auth linkedin
```

This prompts for your Client ID and Secret (first time only), starts a local callback listener on `localhost:9876`, opens your browser for authorization, and saves the token to `.env`.

### Troubleshooting: manual token generation

Use this only if `velocirepo auth linkedin` cannot complete the browser callback flow.

1. Open this URL in your browser (replace `YOUR_CLIENT_ID`):

   ```
   https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:9876/callback&scope=r_organization_social
   ```

2. Authorize the app. You'll be redirected to `http://localhost:9876/callback?code=XXXX`. Copy the `code` value from the URL.

3. Exchange the code for a token:

   ```bash
   curl -s -X POST https://www.linkedin.com/oauth/v2/accessToken \
     -d grant_type=authorization_code \
     -d "code=XXXX" \
     -d "redirect_uri=http://localhost:9876/callback" \
     -d "client_id=YOUR_CLIENT_ID" \
     -d "client_secret=YOUR_CLIENT_SECRET" | jq .access_token -r
   ```

4. Add the token to your `.env` file (next to `velocirepo.toml`):

   ```
   LINKEDIN_TOKEN=AQV...
   ```

## Notes

- Tokens expire after approximately **60 days**. You'll need to re-authenticate when they expire.
- The CLI listens on `localhost:9876` during `velocirepo auth linkedin`; make sure the port is available.
- In the manual troubleshooting flow, the redirect URL (`localhost:9876`) does not need to be a running server — you just need to copy the code from the browser's address bar.
- If you track multiple organizations, only one token is needed (assuming your LinkedIn account has Super Admin access to all of them).
