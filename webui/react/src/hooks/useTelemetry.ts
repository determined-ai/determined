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
  private isReady: boolean;

  constructor() {
    this.isReady = false;
  }

  async load(info: DeterminedInfo): Promise<void> {
    if (!analytics?.load || !analytics?.page || !info.isTelemetryEnabled) return;

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
        this.isReady = true;
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Failed to load telemetry.',
        silent: true,
        type: ErrorType.Server,
      });
    }
  }

  setup(auth: Auth, info: DeterminedInfo) {
    if (!analytics?.identify || !auth.user || !info.isTelemetryEnabled) return;

    /*
     * Segment key should be 32 characters composed of upper case letters,
     * lower case letters and numbers 0-9.
     */
    try {
      analytics.identify(auth.user.id.toString(), {
        clusterId: info.clusterId,
        clusterName: info.clusterName,
        edition: 'OSS',
        masterId: info.masterId,
        username: auth.user.username,
        version: info.version,
      });
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Failed to set telemetry identity.',
        silent: true,
        type: ErrorType.Server,
      });
    }
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
}

// Create the instance outside of the hook to ensure a single instance.
const telemetryInstance = new Telemetry();

interface TelemetryHook {
  loadTelemetry: (info: DeterminedInfo) => void;
  resetTelemetry: () => void;
  setupTelemetry: (auth: Auth, info: DeterminedInfo) => void;
  track: (...args: any[]) => void;
  trackPage: (...args: any[]) => void;
}

const useTelemetry = (): TelemetryHook => {
  const telemetry = useRef<Telemetry>(telemetryInstance);

  return {
    loadTelemetry: telemetry.current.load,
    resetTelemetry: telemetry.current.reset,
    setupTelemetry: telemetry.current.setup,
    track: telemetry.current.track,
    trackPage: telemetry.current.page,
  };
};

export default useTelemetry;
