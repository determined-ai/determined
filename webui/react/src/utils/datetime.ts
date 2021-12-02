import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

import { StartEndTimes } from 'types';

dayjs.extend(utc);

/* eslint-disable @typescript-eslint/no-var-requires */
const humanizeDuration = require('humanize-duration');

export const DEFAULT_DATETIME_FORMAT = 'YYYY-MM-DD, HH:mm:ss';

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
  unitMeasures: {
    d: 86400000,
    h: 3600000,
    m: 60000,
    mo: 2628000000, // Override incorrect default of 2629800000
    ms: 1,
    s: 1000,
    w: 604800000,
    y: 31536000000, // Override incorrect default of 31557600000
  },
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
