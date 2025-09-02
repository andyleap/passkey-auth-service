function showMessage(text, type = 'error') {
    const messageDiv = document.getElementById('message');
    messageDiv.className = type; // Use design system classes: 'success' or 'error'
    messageDiv.textContent = text;
    messageDiv.style.display = 'block';
}

function clearMessage() {
    const messageDiv = document.getElementById('message');
    messageDiv.style.display = 'none';
    messageDiv.className = '';
    messageDiv.textContent = '';
}

async function authenticate() {
    const username = document.getElementById('username').value.trim();
    if (!username) {
        showMessage('Please enter a username');
        return;
    }
    
    const authBtn = document.querySelector('button[onclick="authenticate()"]');
    const regBtn = document.querySelector('button[onclick="register()"]');
    
    // Set loading state
    authBtn.classList.add('btn--loading');
    authBtn.disabled = true;
    regBtn.disabled = true;
    
    clearMessage();
    showMessage('Starting authentication...', 'success');
    
    try {
        // Begin login
        const response = await fetch('/api/v1/login/begin?username=' + encodeURIComponent(username), {
            method: 'POST'
        });
        
        if (!response.ok) {
            throw new Error('Failed to start authentication: ' + response.statusText);
        }
        
        const options = await response.json();
        showMessage('Please use your passkey...', 'success');
        
        // WebAuthn login
        const publicKeyOptions = PublicKeyCredential.parseRequestOptionsFromJSON(options.publicKey);
        const credential = await navigator.credentials.get({
            publicKey: publicKeyOptions
        });
        
        if (!credential) {
            throw new Error('Authentication was cancelled');
        }
        
        showMessage('Completing authentication...', 'success');
        
        // Finish login
        const credentialData = credential.toJSON();
        const verifyResponse = await fetch('/api/v1/login/finish?username=' + encodeURIComponent(username), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(credentialData)
        });
        
        if (!verifyResponse.ok) {
            throw new Error('Authentication failed: ' + verifyResponse.statusText);
        }
        
        const result = await verifyResponse.json();
        showMessage('Authentication successful! Redirecting...', 'success');
        
        // Complete OAuth flow - redirect with authorization code
        completeOAuthFlow(username);
        
    } catch (error) {
        console.error('Authentication error:', error);
        showMessage('Authentication failed: ' + error.message);
    } finally {
        // Reset loading state
        authBtn.classList.remove('btn--loading');
        authBtn.disabled = false;
        regBtn.disabled = false;
    }
}

async function register() {
    const username = document.getElementById('username').value.trim();
    if (!username) {
        showMessage('Please enter a username');
        return;
    }
    
    const authBtn = document.querySelector('button[onclick="authenticate()"]');
    const regBtn = document.querySelector('button[onclick="register()"]');
    
    // Set loading state
    regBtn.classList.add('btn--loading');
    authBtn.disabled = true;
    regBtn.disabled = true;
    
    clearMessage();
    showMessage('Starting registration...', 'success');
    
    try {
        // Begin registration
        const response = await fetch('/api/v1/register/begin?username=' + encodeURIComponent(username), {
            method: 'POST'
        });
        
        if (!response.ok) {
            throw new Error('Failed to start registration: ' + response.statusText);
        }
        
        const options = await response.json();
        showMessage('Please create your passkey...', 'success');
        
        // WebAuthn registration
        const publicKeyOptions = PublicKeyCredential.parseCreationOptionsFromJSON(options.publicKey);
        const credential = await navigator.credentials.create({
            publicKey: publicKeyOptions
        });
        
        if (!credential) {
            throw new Error('Registration was cancelled');
        }
        
        showMessage('Completing registration...', 'success');
        
        // Finish registration
        const credentialData = credential.toJSON();
        const verifyResponse = await fetch('/api/v1/register/finish?username=' + encodeURIComponent(username), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(credentialData)
        });
        
        if (!verifyResponse.ok) {
            throw new Error('Registration failed: ' + verifyResponse.statusText);
        }
        
        showMessage('Registration successful! Now signing you in...', 'success');
        
        // Complete OAuth flow - redirect with authorization code
        completeOAuthFlow(username);
        
    } catch (error) {
        console.error('Registration error:', error);
        showMessage('Registration failed: ' + error.message);
    } finally {
        // Reset loading state
        regBtn.classList.remove('btn--loading');
        authBtn.disabled = false;
        regBtn.disabled = false;
    }
}

async function completeOAuthFlow(username) {
    try {
        // Create authorization code and redirect
        const response = await fetch('/oauth/complete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: username,
                client_id: authData.client_id,
                redirect_uri: authData.redirect_uri,
                state: authData.state
            })
        });
        
        if (!response.ok) {
            throw new Error('Failed to complete OAuth flow');
        }
        
        const result = await response.json();
        
        // Redirect to the callback URL
        window.location.href = result.redirect_url;
        
    } catch (error) {
        console.error('OAuth completion error:', error);
        showMessage('Failed to complete authentication flow: ' + error.message);
    }
}