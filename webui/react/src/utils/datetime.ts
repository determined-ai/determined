import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import humanizeDuration from 'humanize-duration';

import { BulkExperimentItem, StartEndTimes } from 'types';

dayjs.extend(utc);

export const DURATION_SECOND = 1000;
export const DURATION_MINUTE = 60 * DURATION_SECOND;
export const DURATION_HOUR = 60 * DURATION_MINUTE;
export const DURATION_DAY = 24 * DURATION_HOUR;
export const DURATION_WEEK = 7 * DURATION_DAY;
export const DURATION_YEAR = 365 * DURATION_DAY;
export const DURATION_MONTH = DURATION_YEAR / 12;
export const DEFAULT_DATETIME_FORMAT = 'YYYY-MM-DD, HH:mm:ss';
export const DURATION_UNIT_MEASURES = {
  d: DURATION_DAY,
  h: DURATION_HOUR,
  m: DURATION_MINUTE,
  mo: DURATION_MONTH, // Override incorrect default of 2629800000
  ms: 1,
  s: DURATION_SECOND,
  w: DURATION_WEEK,
  y: DURATION_YEAR, // Override incorrect default of 31557600000
};

export const durationInEnglish = humanizeDuration.humanizer({
  conjunction: ' ',
  language: 'shortEn',
  languages: {
    shortEn: {
      d: (): string => 'd',
      h: (): string => 'h',
      m: (): string => 'm',
      mo: (): string => 'mo',
      ms: (): string => 'ms',
      s: (): string => 's',
      w: (): string => 'w',
      y: (): string => 'y',
    },
  },
  largest: 2,
  round: true,
  spacer: '',
  unitMeasures: DURATION_UNIT_MEASURES,
  units: ['y', 'mo', 'w', 'd', 'h', 'm', 's', 'ms'],
});

// Experiment duration (naive) in miliseconds
export const getDuration = (times: StartEndTimes): number => {
  const endTime = times.endTime ? new Date(times.endTime) : new Date();
  const startTime = new Date(times.startTime);
  return endTime.getTime() - startTime.getTime();
};

export const secondToHour = (seconds: number): number => seconds / 3600;

export const getDurationInEnglish = (record: BulkExperimentItem): string => {
  const duration = getDuration(record);
  const options = {
    conjunction: ' ',
    delimiter: ' ',
    largest: 2,
    serialComma: false,
    unitMeasures: { ...DURATION_UNIT_MEASURES, ms: 1000 },
  };
  return durationInEnglish(duration, options);
};

const JUST_NOW = 'Just Now';

const DATE_FORMAT = 'MMM D, YYYY';
export const getTimeInEnglish = (d: Date): string => {
  const options = {
    conjunction: ' ',
    delimiter: ' ',
    largest: 1,
    serialComma: false,
  };

  const now = Date.now();
  const milliseconds = d.valueOf();
  const delta = milliseconds === undefined ? 0 : now - milliseconds;

  if (delta < DURATION_MINUTE) {
    return JUST_NOW;
  } else if (delta >= DURATION_YEAR) {
    return dayjs(milliseconds).format(DATE_FORMAT);
  } else {
    return `${durationInEnglish(delta, options)} ago`;
  }
};
