// Main TypeScript entry point for devrig website
import React from 'react';
import ReactDOM from 'react-dom/client';
import { PlatformDetector } from './components/PlatformDetector';
import { DownloadButtons } from './components/DownloadButtons';

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  console.log('devrig website loaded');

  // Mount React components
  const platformDetectorEl = document.getElementById('platform-detector');
  if (platformDetectorEl) {
    const root = ReactDOM.createRoot(platformDetectorEl);
    root.render(<PlatformDetector />);
  }

  const downloadButtonsEl = document.getElementById('download-buttons');
  if (downloadButtonsEl) {
    const root = ReactDOM.createRoot(downloadButtonsEl);
    root.render(<DownloadButtons />);
  }
});
