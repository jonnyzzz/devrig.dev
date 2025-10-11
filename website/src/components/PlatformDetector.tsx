import React, { useState, useEffect } from 'react';

interface Platform {
  os: string;
  arch: string;
  name: string;
}

export const PlatformDetector: React.FC = () => {
  const [platform, setPlatform] = useState<Platform | null>(null);

  useEffect(() => {
    const detected = detectPlatform();
    setPlatform(detected);
  }, []);

  const detectPlatform = (): Platform | null => {
    const userAgent = navigator.userAgent.toLowerCase();
    const platform = navigator.platform.toLowerCase();

    let os = 'unknown';
    let osName = 'Unknown';

    if (platform.includes('win')) {
      os = 'windows';
      osName = 'Windows';
    } else if (platform.includes('mac')) {
      os = 'darwin';
      osName = 'macOS';
    } else if (platform.includes('linux')) {
      os = 'linux';
      osName = 'Linux';
    }

    // Detect architecture
    let arch = 'x86_64';
    if (userAgent.includes('arm') || userAgent.includes('aarch64')) {
      arch = 'arm64';
    }

    if (os === 'unknown') return null;

    return {
      os,
      arch,
      name: `${osName} ${arch === 'arm64' ? 'ARM64' : 'x86-64'}`
    };
  };

  if (!platform) {
    return (
      <div className="platform-detector">
        <p>Detecting your platform...</p>
      </div>
    );
  }

  return (
    <div className="platform-detector">
      <div className="detected-platform">
        <h3>Your Platform</h3>
        <p className="platform-name">
          <strong>{platform.name}</strong>
        </p>
        <p className="platform-info">
          We've detected you're running <code>{platform.os}</code> on <code>{platform.arch}</code> architecture.
        </p>
      </div>
    </div>
  );
};
