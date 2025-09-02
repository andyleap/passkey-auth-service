# OAuth-Style Integration Guide

The Passkey Authentication Service now supports OAuth-style redirect flows, making it easy to integrate with any web application.

## 🔄 OAuth Flow Overview

1. **Redirect to Auth Service**: Your app redirects users to the auth service
2. **Beautiful Authentication UI**: Users authenticate with their passkey
3. **Redirect Back**: Users are redirected back to your app with an authorization code
4. **Exchange Code**: Your app exchanges the code for user information

## 🚀 Quick Integration

### Step 1: Redirect to Authorization Endpoint

Redirect users to:
```
https://your-auth-service.com/authorize?client_id=YOUR_CLIENT_ID&redirect_uri=YOUR_CALLBACK_URL&state=RANDOM_STATE
```

**Parameters:**
- `client_id`: Your application identifier (e.g., "demo-app")
- `redirect_uri`: Where to redirect after authentication
- `state`: Random string to prevent CSRF (optional but recommended)

### Step 2: Handle the Callback

After successful authentication, users are redirected to your `redirect_uri` with:
```
https://your-app.com/callback?code=AUTHORIZATION_CODE&state=YOUR_STATE
```

**Parameters:**
- `code`: Authorization code to exchange for user info
- `state`: The same state value you sent (verify this matches)

### Step 3: Exchange Code for User Info

Make a POST request to exchange the authorization code:

```javascript
const response = await fetch('https://your-auth-service.com/oauth/token', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    code: authorizationCode,
    client_id: 'your-client-id',
    redirect_uri: 'https://your-app.com/callback'
  })
});

const userInfo = await response.json();
// Returns: { username, user_id, client_id, expires_at }
```

## 🎨 User Experience

Users see a beautiful, modern authentication interface with:
- Clean, gradient design
- Clear messaging about which app is requesting access
- Simple passkey authentication
- Option to register new passkeys
- Smooth redirects back to your application

## ⚙️ Configuration

### Demo Clients (Pre-configured)

The service comes with demo clients configured:

- **demo-app**: For development and testing
  - Redirect URIs: `http://localhost:3000/callback`, `https://localhost:3000/callback`
- **test-app**: Additional test client
  - Redirect URIs: `http://localhost:3001/callback`, `https://localhost:3001/callback`

### Adding Your Own Client

Currently, clients are configured in code (`internal/oauth/oauth.go`). In a production deployment, you'd typically:

1. Add your client to the `clients` map
2. Configure allowed redirect URIs
3. Or implement a database-backed client store

Example client configuration:
```go
"your-app": {
    ID:   "your-app",
    Name: "Your Application Name",
    RedirectURIs: []string{
        "https://your-app.com/callback",
        "https://staging.your-app.com/callback",
    },
    CreatedAt: time.Now(),
}
```

## 🧪 Testing with Demo Client

1. **Start the auth service:**
   ```bash
   make run
   ```

2. **Open the demo client:**
   Open `examples/demo-client.html` in your browser

3. **Test the flow:**
   - Click "Login with Passkey"
   - Authenticate on the auth service
   - See the successful redirect with authorization code
   - Exchange code for user information

## 🔗 Integration Examples

### React/Next.js
```javascript
// Redirect to auth service
const handleLogin = () => {
  const state = generateRandomState();
  localStorage.setItem('oauth_state', state);
  
  const authUrl = `https://auth-service.com/authorize?` +
    `client_id=your-app&` +
    `redirect_uri=${encodeURIComponent(window.location.origin + '/callback')}&` +
    `state=${state}`;
  
  window.location.href = authUrl;
};

// Handle callback (in your /callback route)
const handleCallback = async () => {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  const state = params.get('state');
  
  // Verify state
  if (state !== localStorage.getItem('oauth_state')) {
    throw new Error('Invalid state');
  }
  
  // Exchange code for user info
  const response = await fetch('https://auth-service.com/oauth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      code,
      client_id: 'your-app',
      redirect_uri: window.location.origin + '/callback'
    })
  });
  
  const user = await response.json();
  // Handle successful authentication
};
```

### Express.js
```javascript
app.get('/login', (req, res) => {
  const state = generateRandomState();
  req.session.oauth_state = state;
  
  const authUrl = `https://auth-service.com/authorize?` +
    `client_id=your-app&` +
    `redirect_uri=${encodeURIComponent(req.protocol + '://' + req.get('host') + '/callback')}&` +
    `state=${state}`;
  
  res.redirect(authUrl);
});

app.get('/callback', async (req, res) => {
  const { code, state } = req.query;
  
  // Verify state
  if (state !== req.session.oauth_state) {
    return res.status(400).send('Invalid state');
  }
  
  // Exchange code
  const response = await fetch('https://auth-service.com/oauth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      code,
      client_id: 'your-app',
      redirect_uri: `${req.protocol}://${req.get('host')}/callback`
    })
  });
  
  const user = await response.json();
  req.session.user = user;
  res.redirect('/dashboard');
});
```

## 🔐 Security Considerations

1. **Always verify the state parameter** to prevent CSRF attacks
2. **Use HTTPS** for all redirect URIs in production
3. **Authorization codes expire in 10 minutes** - exchange them quickly
4. **Validate redirect URIs** - only pre-configured URIs are allowed
5. **Store user sessions securely** after authentication

## 🎯 Benefits of This Approach

- **No complex WebAuthn implementation** in your app
- **Beautiful, consistent authentication UI** across all your apps
- **Centralized user management** - users can use the same passkey everywhere
- **Easy integration** - just redirect and handle callbacks
- **OAuth-like flow** that developers are familiar with
- **Secure by design** with proper authorization code flow

## 🚀 Production Deployment

For production use:
1. Configure proper SSL certificates
2. Set up client management (database-backed)
3. Configure appropriate redirect URIs for your domains
4. Set up monitoring and logging
5. Consider rate limiting on authorization endpoints

The service is now ready to act as a centralized authentication provider for all your applications!