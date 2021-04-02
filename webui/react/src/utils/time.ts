import { StartEndTimes } from 'types';

/* eslint-disable @typescript-eslint/no-var-requires */
const humanizeDuration = require('humanize-duration');

// Experiment duration (naive) in miliseconds
export const getDuration = (times: StartEndTimes): number => {
  const endTime = times.endTime ? new Date(times.endTime) : new Date();
  const startTime = new Date(times.startTime);
  return endTime.getTime() - startTime.getTime();
};

export const shortEnglishHumannizer = humanizeDuration.humanizer({
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
});

export const secondToHour = (seconds: number): number => {
  const hourInSecond = (60 * 60);
  return (seconds / hourInSecond);
};
