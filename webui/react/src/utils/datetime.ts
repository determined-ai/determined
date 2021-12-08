import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

import { StartEndTimes } from 'types';

dayjs.extend(utc);

/* eslint-disable @typescript-eslint/no-var-requires */
const humanizeDuration = require('humanize-duration');

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
  units: [ 'y', 'mo', 'w', 'd', 'h', 'm', 's', 'ms' ],
});

export const formatDatetime = (
  datetime: string,
  options: { format?: string, inputUTC?: boolean, outputUTC?: boolean, } = {},
): string => {
  const config = {
    format: DEFAULT_DATETIME_FORMAT,
    inputUTC: false,
    outputUTC: true,
    ...options,
  };
  // Strip out the timezone info if we want to force UTC input.
  const dateString = config.inputUTC ? stripTimezone(datetime) : datetime;

  // `dayjs.utc` respects timezone in the datetime string if available.
  let dayjsDate = dayjs.utc(dateString);

  // Prep the date as UTC or local time based on output UTC option.
  if (!config.outputUTC) dayjsDate = dayjsDate.local();

  // Return the formatted date based on provided format.
  return dayjsDate.format(config.format);
};

// Experiment duration (naive) in miliseconds
export const getDuration = (times: StartEndTimes): number => {
  const endTime = times.endTime ? new Date(times.endTime) : new Date();
  const startTime = new Date(times.startTime);
  return endTime.getTime() - startTime.getTime();
};

export const secondToHour = (seconds: number): number => seconds / 3600;

export const stripTimezone = (datetime: string): string => {
  const timezoneRegex = /(Z|(-|\+)\d{2}:\d{2})$/;
  return datetime.replace(timezoneRegex, '');
};
