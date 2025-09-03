import { useState } from 'preact/hooks';
import { apiRequest } from '../utils/api.js';

export function CredentialsSection({ credentials, username, loading, onRefresh }) {
    const [addingPasskey, setAddingPasskey] = useState(false);
    const [error, setError] = useState(null);

    const addPasskey = async () => {
        if (!username || username === 'Loading...') {
            setError('Unable to determine username. Please refresh the page.');
            return;
        }
        
        try {
            setAddingPasskey(true);
            setError(null);
            
            // Begin registration
            const response = await apiRequest('/api/v1/register/begin?username=' + encodeURIComponent(username), {
                method: 'POST'
            });
            
            if (!response || !response.ok) {
                throw new Error('Failed to start passkey creation: ' + response.statusText);
            }
            
            const options = await response.json();
            
            // WebAuthn registration
            const publicKeyOptions = PublicKeyCredential.parseCreationOptionsFromJSON(options.publicKey);
            const credential = await navigator.credentials.create({
                publicKey: publicKeyOptions
            });
            
            if (!credential) {
                throw new Error('Passkey creation was cancelled');
            }
            
            // Finish registration
            const credentialData = credential.toJSON();
            const verifyResponse = await apiRequest('/api/v1/register/finish?username=' + encodeURIComponent(username), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(credentialData)
            });
            
            if (!verifyResponse || !verifyResponse.ok) {
                throw new Error('Passkey creation failed: ' + verifyResponse.statusText);
            }
            
            // Reload credentials to show the new passkey
            onRefresh();
            
        } catch (error) {
            console.error('Add passkey error:', error);
            setError('Failed to add passkey: ' + error.message);
        } finally {
            setAddingPasskey(false);
        }
    };

    const deleteCredential = async (credentialId) => {
        if (!confirm('Are you sure you want to delete this passkey? You won\'t be able to sign in with it anymore.')) {
            return;
        }
        
        try {
            const response = await apiRequest(`/api/v1/user/credentials/${encodeURIComponent(credentialId)}`, {
                method: 'DELETE'
            });
            
            if (!response) return;
            
            if (!response.ok) {
                throw new Error(`Failed to delete credential: ${response.statusText}`);
            }
            
            // Reload credentials
            onRefresh();
        } catch (error) {
            alert('Failed to delete passkey: ' + error.message);
        }
    };

    return (
        <div class="panel-section">
            <div class="section-header">
                <h2 class="section-title">
                    🔑 Your Passkeys
                </h2>
                <div>
                    <button 
                        class="refresh-btn" 
                        onClick={onRefresh} 
                        title="Refresh"
                        disabled={loading}
                    >
                        🔄
                    </button>
                    <button 
                        class="btn btn--primary btn--sm" 
                        onClick={addPasskey}
                        disabled={addingPasskey}
                    >
                        {addingPasskey ? 'Creating...' : 'Add Passkey'}
                    </button>
                </div>
            </div>
            <div class="section-content">
                {error && (
                    <div class="error" style="margin-bottom: var(--space-4);">
                        {error}
                        <button 
                            class="btn btn--sm" 
                            onClick={() => setError(null)}
                            style="margin-left: var(--space-2);"
                        >
                            Dismiss
                        </button>
                    </div>
                )}
                
                {loading ? (
                    <div class="loading">
                        {addingPasskey ? 'Creating new passkey...' : 'Loading your passkeys...'}
                    </div>
                ) : credentials.length === 0 ? (
                    <div class="empty-state">
                        <div class="empty-icon">🔑</div>
                        <p>No passkeys found</p>
                        <button 
                            class="btn btn--primary" 
                            onClick={addPasskey}
                            disabled={addingPasskey}
                        >
                            {addingPasskey ? 'Creating...' : 'Add Your First Passkey'}
                        </button>
                    </div>
                ) : (
                    <div class="item-list">
                        {credentials.map((cred, index) => (
                            <div key={cred.id || index} class="item">
                                <div class="item-info">
                                    <div class="item-title">Passkey #{index + 1}</div>
                                    <div class="item-subtitle">
                                        Created: {new Date(cred.createdAt).toLocaleDateString()}
                                    </div>
                                </div>
                                <div class="item-actions">
                                    <button 
                                        class="btn btn--danger btn--sm" 
                                        onClick={() => deleteCredential(cred.id)}
                                        disabled={credentials.length === 1}
                                        title={credentials.length === 1 ? "Can't delete your last passkey" : "Delete this passkey"}
                                    >
                                        Delete
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}