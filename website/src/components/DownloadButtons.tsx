import React, { useState, useEffect } from 'react';

interface Release {
  os: string;
  arch: string;
  url: string;
  sha512: string;
}

interface LatestRelease {
  version: string;
  releaseDate: string;
  releases: Release[];
}

export const DownloadButtons: React.FC = () => {
  const [releases, setReleases] = useState<LatestRelease | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/download/latest.json')
      .then(res => res.json())
      .then(data => {
        setReleases(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Failed to load releases:', err);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <div className="download-buttons">Loading releases...</div>;
  }

  if (!releases) {
    return <div className="download-buttons">Failed to load releases</div>;
  }

  const osNames: Record<string, string> = {
    'linux': 'Linux',
    'darwin': 'macOS',
    'windows': 'Windows'
  };

  const archNames: Record<string, string> = {
    'x86_64': 'x86-64',
    'arm64': 'ARM64'
  };

  return (
    <div className="download-buttons">
      <h3>Quick Downloads</h3>
      <div className="button-grid">
        {releases.releases.map((release, idx) => {
          const osName = osNames[release.os] || release.os;
          const archName = archNames[release.arch] || release.arch;
          const fileName = release.url.split('/').pop();

          return (
            <a
              key={idx}
              href={release.url.replace('https://devrig.dev', '')}
              className="download-button"
              download={fileName}
            >
              <div className="button-icon">📦</div>
              <div className="button-text">
                <div className="button-title">{osName}</div>
                <div className="button-subtitle">{archName}</div>
              </div>
            </a>
          );
        })}
      </div>
      <p className="version-info">Version: {releases.version}</p>
    </div>
  );
};
