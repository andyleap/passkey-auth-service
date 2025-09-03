import { toggleTheme, apiRequest } from '../utils/api.js';

export function Header({ username }) {
    const handleLogout = async () => {
        if (confirm('Are you sure you want to logout?')) {
            try {
                await apiRequest('/api/v1/logout', { method: 'POST' });
            } catch (error) {
                console.error('Logout error:', error);
            }
            // Clear the session cookie and redirect to landing page
            document.cookie = 'session_id=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
            window.location.href = '/';
        }
    };

    return (
        <header class="header">
            <div class="header-info">
                <div class="logo">🔐</div>
                <div>
                    <div class="username">{username}</div>
                    <div style="font-size: var(--text-sm); color: var(--color-text-subtle);">
                        Control Panel
                    </div>
                </div>
            </div>
            <div class="controls">
                <button 
                    class="theme-toggle theme-toggle--header" 
                    onClick={toggleTheme} 
                    title="Toggle Theme"
                >
                    <span class="light-only">🌙</span>
                    <span class="dark-only">☀️</span>
                </button>
                <button class="btn btn--secondary" onClick={handleLogout}>
                    Logout
                </button>
            </div>
        </header>
    );
}