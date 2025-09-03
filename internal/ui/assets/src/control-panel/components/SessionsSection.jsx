import { apiRequest } from '../utils/api.js';

export function SessionsSection({ sessions, loading, onRefresh }) {
    const deleteSession = async (sessionId) => {
        if (!confirm('Are you sure you want to end this session?')) {
            return;
        }
        
        try {
            const response = await apiRequest(`/api/v1/user/sessions/${sessionId}`, {
                method: 'DELETE'
            });
            
            if (!response) return;
            
            if (!response.ok) {
                throw new Error(`Failed to delete session: ${response.statusText}`);
            }
            
            // Reload sessions
            onRefresh();
        } catch (error) {
            alert('Failed to end session: ' + error.message);
        }
    };

    return (
        <div class="panel-section">
            <div class="section-header">
                <h2 class="section-title">
                    🔐 Active Sessions
                </h2>
                <button 
                    class="refresh-btn" 
                    onClick={onRefresh} 
                    title="Refresh"
                    disabled={loading}
                >
                    🔄
                </button>
            </div>
            <div class="section-content">
                {loading ? (
                    <div class="loading">Loading your sessions...</div>
                ) : sessions.length === 0 ? (
                    <div class="empty-state">
                        <div class="empty-icon">🔐</div>
                        <p>No active sessions</p>
                    </div>
                ) : (
                    <div class="item-list">
                        {sessions.map((session) => (
                            <div key={session.id} class="item">
                                <div class="item-info">
                                    <div class="item-title">
                                        Session {session.id.substring(0, 8)}...
                                        {session.current && (
                                            <span class="current-badge" style="margin-left: var(--space-2);">
                                                Current
                                            </span>
                                        )}
                                    </div>
                                    <div class="item-subtitle">
                                        Created: {new Date(session.createdAt).toLocaleString()} | 
                                        Expires: {new Date(session.expiresAt).toLocaleString()}
                                    </div>
                                </div>
                                <div class="item-actions">
                                    {!session.current && (
                                        <button 
                                            class="btn btn--danger btn--sm" 
                                            onClick={() => deleteSession(session.id)}
                                        >
                                            End Session
                                        </button>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}