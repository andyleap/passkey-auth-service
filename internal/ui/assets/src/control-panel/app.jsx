import { useState, useEffect } from 'preact/hooks';
import { CredentialsSection } from './components/CredentialsSection.jsx';
import { SessionsSection } from './components/SessionsSection.jsx';
import { Header } from './components/Header.jsx';
import { apiRequest } from './utils/api.js';

export function App() {
    const [user, setUser] = useState({ username: 'Loading...', credentials: [], sessions: [] });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const loadUserData = async () => {
        try {
            setLoading(true);
            setError(null);
            
            const [credentialsResponse, sessionsResponse] = await Promise.all([
                apiRequest('/api/v1/user/credentials'),
                apiRequest('/api/v1/user/sessions')
            ]);

            if (!credentialsResponse || !sessionsResponse) {
                return; // apiRequest handles redirects
            }

            if (!credentialsResponse.ok) {
                throw new Error(`Failed to load credentials: ${credentialsResponse.statusText}`);
            }
            if (!sessionsResponse.ok) {
                throw new Error(`Failed to load sessions: ${sessionsResponse.statusText}`);
            }

            const credentialsData = await credentialsResponse.json();
            const sessionsData = await sessionsResponse.json();

            setUser({
                username: credentialsData.username,
                credentials: credentialsData.credentials || [],
                sessions: sessionsData.sessions || []
            });
        } catch (err) {
            console.error('Failed to load user data:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const refreshCredentials = async () => {
        try {
            const response = await apiRequest('/api/v1/user/credentials');
            if (response && response.ok) {
                const data = await response.json();
                setUser(prev => ({
                    ...prev,
                    credentials: data.credentials || []
                }));
            }
        } catch (err) {
            console.error('Failed to refresh credentials:', err);
        }
    };

    const refreshSessions = async () => {
        try {
            const response = await apiRequest('/api/v1/user/sessions');
            if (response && response.ok) {
                const data = await response.json();
                setUser(prev => ({
                    ...prev,
                    sessions: data.sessions || []
                }));
            }
        } catch (err) {
            console.error('Failed to refresh sessions:', err);
        }
    };

    useEffect(() => {
        loadUserData();
    }, []);

    if (error) {
        return (
            <div class="control-panel-body">
                <div class="main-container">
                    <div class="error">
                        Failed to load control panel: {error}
                        <button class="btn btn--sm" onClick={loadUserData}>
                            Retry
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div class="control-panel-body">
            <Header username={user.username} />
            
            <div class="main-container">
                <CredentialsSection 
                    credentials={user.credentials}
                    username={user.username}
                    loading={loading}
                    onRefresh={refreshCredentials}
                />
                
                <SessionsSection 
                    sessions={user.sessions}
                    loading={loading}
                    onRefresh={refreshSessions}
                />
            </div>
        </div>
    );
}