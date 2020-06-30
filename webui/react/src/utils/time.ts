import { Experiment } from 'types';

/* eslint-disable @typescript-eslint/no-var-requires */
const humanizeDuration = require('humanize-duration');

// Experiment duration (naive) in miliseconds
export const experimentDuration = (experiment: Experiment): number => {
  const endTime = experiment.endTime ? new Date(experiment.endTime) : new Date();
  const startTime = new Date(experiment.startTime);
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
