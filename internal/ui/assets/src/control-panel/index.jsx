import { render } from 'preact';
import { App } from './app.jsx';
import { loadSavedTheme } from './utils/api.js';

// Load saved theme on startup
loadSavedTheme();

// Render the app
render(<App />, document.body);