/* eslint-disable @typescript-eslint/no-explicit-any */
import { useRef } from 'react';

import { getTelemetry } from 'services/api';
import { Auth, DeterminedInfo } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

/*
 * Telemetry is written as a modular class instance instead of
 * a React hook because the telemetry capabilities are needed
 * outside of React.
 */
class Telemetry {
  isLoaded: boolean;
  isIdentified: boolean;

  constructor() {
    this.isLoaded = false;
    this.isIdentified = false;
  }

  async update(auth: Auth, info: DeterminedInfo) {
    if (!info.isTelemetryEnabled) return;

    // Attempt to load analytics first.
    await this.load(info);

    // Update identify if necessary.
    this.identify(auth, info);
  }

  reset() {
    if (analytics?.reset) analytics.reset();
  }

  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  page(...args: any[]) {
    if (analytics?.page) analytics.page(...args);
  }

  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  track(event: string, ...args: any[]) {
    if (analytics?.track) analytics.track(event, ...args);
  }

  async load(info: DeterminedInfo): Promise<void> {
    if (this.isLoaded || !analytics?.load || !analytics?.page || !info.isTelemetryEnabled) return;

    /*
     * Segment key should be 32 characters composed of upper case letters,
     * lower case letters and numbers 0-9.
     */
    try {
      const telemetry = await getTelemetry({});
      const isProperKey = telemetry.segmentKey && /^[a-z0-9]{32}$/i.test(telemetry.segmentKey);
      if (isProperKey) {
        analytics.load(telemetry.segmentKey || '');
        analytics.page();
        this.isLoaded = true;
      }
    } catch (e) {
      handleError({
        level: ErrorLevel.Error,
        publicMessage: 'Failed to load telemetry.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }

  identify(auth: Auth, info: DeterminedInfo) {
    if (!this.isLoaded || !analytics?.identify) return;

    /*
     * Segment key should be 32 characters composed of upper case letters,
     * lower case letters and numbers 0-9.
     */
    try {
      if (!this.isIdentified && auth.isAuthenticated && auth.user) {
        analytics.identify(auth.user.id.toString(), {
          clusterId: info.clusterId,
          clusterName: info.clusterName,
          edition: 'OSS',
          masterId: info.masterId,
          username: auth.user.username,
          version: info.version,
        });
        this.isIdentified = true;
      } else if (this.isIdentified && !auth.isAuthenticated) {
        this.reset();
        this.isIdentified = false;
      }
    } catch (e) {
      handleError({
        level: ErrorLevel.Error,
        publicMessage: 'Failed to set telemetry identity.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }
}

// Create the instance outside of the hook to ensure a single instance.
export const telemetryInstance = new Telemetry();

interface TelemetryHook {
  track: (...args: any[]) => void;
  trackPage: (...args: any[]) => void;
  updateTelemetry: (auth: Auth, info: DeterminedInfo) => void;
}

const useTelemetry = (): TelemetryHook => {
  const telemetry = useRef<Telemetry>(telemetryInstance);

  return {
    track: telemetry.current.track.bind(telemetry.current),
    trackPage: telemetry.current.page.bind(telemetry.current),
    updateTelemetry: telemetry.current.update.bind(telemetry.current),
  };
};

export default useTelemetry;
