import { DeterminedInfo } from 'types';

interface InternalSegmentAnalytics extends SegmentAnalytics.AnalyticsJS {
  methods: string[];
}

interface AnalyticsData {
  analytics?: SegmentAnalytics.AnalyticsJS;
  isEnabled: boolean;
}

/*
 * This object is stored outside of the React space to allow analytics to be
 * accessible outside of React components, such as `ErrorHandler`.
 */
const data: AnalyticsData = {
  analytics: undefined,
  isEnabled: false,
};

/*
 * Segment analytics library is loaded on `index.html` dynamically through a JS
 * snippet. The library has a `methods` string array which is an inventory of
 * methods to load. Though there is a `ready` callback, it is not reliably called
 * and it is lacking a boolean ready state. Instead when all the methods are defined
 * on the library object `window.analytics` it is considered ready.
 */
const getReadyAnalytics = (): SegmentAnalytics.AnalyticsJS | undefined => {
  if (data.analytics) return data.analytics;

  const analytics: InternalSegmentAnalytics = window.analytics;
  if (analytics) {
    const keys = Object.keys(analytics).reduce((acc, key) => {
      acc[key] = true;
      return acc;
    }, {} as Record<string, boolean>);

    const methods: string[] = [ ...(analytics.methods || []), 'identify', 'load', 'page' ];
    const missingMethods = methods.some(method => keys[method] === undefined);
    if (!missingMethods) {
      data.analytics = analytics;
      return data.analytics;
    }
  }
  return undefined;
};

export const setupAnalytics = (info: DeterminedInfo): void => {
  if (!data.analytics) data.analytics = getReadyAnalytics();
  if (!data.analytics || data.isEnabled) return;

  /*
   * Segment key should be 32 characters composed of upper case letters,
   * lower case letters and numbers 0-9.
   */
  const telemetry = info.telemetry;
  const isEnabled = telemetry.enabled;
  const isProperKey = telemetry.segmentKey && /^[a-z0-9]{32}$/i.test(telemetry.segmentKey);
  if (isEnabled && isProperKey) {
    data.analytics.load(telemetry.segmentKey || '');
    data.analytics.identify(info.clusterId);
    data.isEnabled = true;
  }
};

/*
 * Return the analytics library if it is ready and enabled via telemetry config.
 */
export const getAnalytics = (): SegmentAnalytics.AnalyticsJS | undefined => {
  if (data.analytics && data.isEnabled) return data.analytics;
  return undefined;
};

export const recordPageAccess = (pathname: string): void => {
  if (data.analytics && data.isEnabled) data.analytics.page(pathname);
};
