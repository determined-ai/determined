// @ts-ignore
import humanizeDuration from 'humanize-duration';

import { Experiment } from 'types';

// Experiment duration (naive) in miliseconds
export const experimentDuration = (experiment: Experiment): number => {
  const endTime = experiment.endTime ? new Date(experiment.endTime) : new Date();
  const startTime = new Date(experiment.startTime);
  return endTime.getTime() - startTime.getTime();
};

export const shortEnglishHumannizer = humanizeDuration.humanizer({ language: 'shortEn',
  languages: {
    shortEn: {
      y: () => 'y',
      mo: () => 'mo',
      w: () => 'w',
      d: () => 'd',
      h: () => 'h',
      m: () => 'm',
      s: () => 's',
      ms: () => 'ms',
    },
  },
  round: true,
  conjunction: ' ',
  spacer: '',
});
