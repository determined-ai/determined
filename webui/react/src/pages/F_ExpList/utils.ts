import dayjs from 'dayjs';

import { ExperimentItem } from 'types';
import {
  DURATION_MINUTE,
  DURATION_UNIT_MEASURES,
  DURATION_YEAR,
  durationInEnglish,
  getDuration,
} from 'utils/datetime';

export const getDurationInEnglish = (record: ExperimentItem): string => {
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
