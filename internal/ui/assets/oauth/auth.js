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

function ready(fn) {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', fn);
    } else {
        fn();
    }
}

async function signInWithPasskey() {
    const username = document.getElementById('username').value.trim();
    if (!username) {
        showMessage('Please enter a username');
        return;
    }
    
    const signInBtn = document.getElementById('signin-btn');
    
    // Set loading state
    signInBtn.classList.add('btn--loading');
    signInBtn.disabled = true;
    
    clearMessage();
    showMessage('Checking account...', 'success');
    
    try {
        // Try login first
        const loginResponse = await fetch('/api/v1/login/begin?username=' + encodeURIComponent(username), {
            method: 'POST'
        });
        
        if (loginResponse.ok) {
            // User exists, proceed with login
            await handleLogin(username, loginResponse);
        } else {
            // User doesn't exist, proceed with registration
            showMessage('Creating new passkey...', 'success');
            await handleRegistration(username);
        }
        
    } catch (error) {
        console.error('Sign in error:', error);
        showMessage('Sign in failed: ' + error.message);
    } finally {
        // Reset loading state
        signInBtn.classList.remove('btn--loading');
        signInBtn.disabled = false;
    }
}

async function handleLogin(username, loginResponse) {
    const options = await loginResponse.json();
    showMessage('Please use your passkey to sign in...', 'success');
    
    // WebAuthn login
    const publicKeyOptions = PublicKeyCredential.parseRequestOptionsFromJSON(options.publicKey);
    const credential = await navigator.credentials.get({
        publicKey: publicKeyOptions
    });
    
    if (!credential) {
        throw new Error('Sign in was cancelled');
    }
    
    showMessage('Completing sign in...', 'success');
    
    // Finish login
    const credentialData = credential.toJSON();
    const verifyResponse = await fetch('/api/v1/login/finish?username=' + encodeURIComponent(username), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentialData)
    });
    
    if (!verifyResponse.ok) {
        throw new Error('Sign in failed: ' + verifyResponse.statusText);
    }
    
    showMessage('Sign in successful! Redirecting...', 'success');
    
    // Save username for future use
    localStorage.setItem('passkey-username', username);
    
    completeOAuthFlow(username);
}

async function handleRegistration(username) {
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
        throw new Error('Passkey creation was cancelled');
    }
    
    showMessage('Completing setup...', 'success');
    
    // Finish registration
    const credentialData = credential.toJSON();
    const verifyResponse = await fetch('/api/v1/register/finish?username=' + encodeURIComponent(username), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentialData)
    });
    
    if (!verifyResponse.ok) {
        throw new Error('Passkey creation failed: ' + verifyResponse.statusText);
    }
    
    showMessage('Passkey created! Signing you in...', 'success');
    
    // Save username for future use
    localStorage.setItem('passkey-username', username);
    
    completeOAuthFlow(username);
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

// Initialize event listeners when DOM is ready
ready(function() {
    const signInBtn = document.getElementById('signin-btn');
    const usernameInput = document.getElementById('username');
    
    // Load saved username
    const savedUsername = localStorage.getItem('passkey-username');
    if (savedUsername && usernameInput) {
        usernameInput.value = savedUsername;
    }
    
    signInBtn?.addEventListener('click', signInWithPasskey);
});