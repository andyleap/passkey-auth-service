// API helper for control panel
export async function apiRequest(url, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };
    
    const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'same-origin' // Include cookies
    });
    
    if (response.status === 401) {
        // Session expired, redirect to landing page
        window.location.href = '/';
        return null;
    }
    
    return response;
}

// Theme management
export function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('passkey-theme', newTheme);
}

export function loadSavedTheme() {
    const savedTheme = localStorage.getItem('passkey-theme');
    if (savedTheme) {
        document.documentElement.setAttribute('data-theme', savedTheme);
    }
}