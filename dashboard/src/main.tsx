import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";

// Global error handler to catch unhandled rejections and errors
window.addEventListener('error', (event) => {
  console.error('[Global error]', event.error?.message || event.message, event.error?.stack);
  // Send to backend for remote debugging
  try {
    const body = JSON.stringify({
      message: event.error?.message || event.message || '',
      stack: event.error?.stack || '',
      url: window.location.href,
      userAgent: navigator.userAgent,
    });
    navigator.sendBeacon?.('/api/log/client-error', body);
  } catch { /* best effort */ }
});

window.addEventListener('unhandledrejection', (event) => {
  console.error('[Unhandled rejection]', event.reason?.message || event.reason);
  try {
    const body = JSON.stringify({
      message: event.reason?.message || String(event.reason),
      stack: event.reason?.stack || '',
      url: window.location.href,
      userAgent: navigator.userAgent,
    });
    navigator.sendBeacon?.('/api/log/client-error', body);
  } catch { /* best effort */ }
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
