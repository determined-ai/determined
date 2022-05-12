import { Tooltip } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import React, { useEffect, useMemo, useState } from 'react';

import { isNumber, isString } from 'shared/utils/data';
import {
  DURATION_DAY, DURATION_HOUR, DURATION_MINUTE, DURATION_SECOND,
  DURATION_YEAR, durationInEnglish,
} from 'shared/utils/datetime';
import { capitalize, capitalizeWord } from 'shared/utils/string';

interface Props {
  className?: string;
  dateFormat?: string;
  datetime: Dayjs | Date | number | string;
  long?: boolean;
  noUpdate?: boolean;
  stringCase?: TimeAgoCase;
  units?: number;
}

export enum TimeAgoCase {
  Lower = 'lower',
  Sentence = 'sentence',
  Title = 'title',
}

export const JUST_NOW = 'Just Now';
export const TOOLTIP_FORMAT = 'MMM D, YYYY - h:mm a';

const TimeAgo: React.FC<Props> = ({
  className,
  dateFormat = 'MMM D, YYYY',
  datetime,
  long = false,
  noUpdate = false,
  stringCase = TimeAgoCase.Sentence,
  units = 1,
}: Props) => {
  const [ now, setNow ] = useState(() => Date.now());
  const classes: string[] = [];

  if (className) classes.push(className);

  const milliseconds = useMemo(() => {
    if (isNumber(datetime)) {
      return datetime * (datetime < 10000000000 ? 1000 : 1);
    } else if (isString(datetime)) {
      return new Date(datetime).valueOf();
    } else if ('valueOf' in datetime) {
      return datetime.valueOf();
    }
    return undefined;
  }, [ datetime ]);

  const delta = useMemo(() => {
    return milliseconds === undefined ? 0 : now - milliseconds;
  }, [ milliseconds, now ]);

  const duration = useMemo(() => {
    if (delta < DURATION_MINUTE) return JUST_NOW;
    if (delta >= DURATION_YEAR) return dayjs(milliseconds).format(dateFormat);

    const options = {
      conjunction: ' ',
      delimiter: ' ',
      language: long ? 'en' : 'shortEn',
      largest: units,
      serialComma: false,
      spacer: long ? ' ' : '',
    };
    const time = durationInEnglish(delta, options);
    return `${time} ago`;
  }, [ delta, dateFormat, long, milliseconds, units ]);

  const durationString = useMemo(() => {
    switch (stringCase) {
      case TimeAgoCase.Lower:
        return duration.toLowerCase();
      case TimeAgoCase.Sentence:
        return capitalizeWord(duration);
      case TimeAgoCase.Title:
        return capitalize(duration);
      default:
        return duration;
    }
  }, [ duration, stringCase ]);

  const updateInterval = useMemo(() => {
    if (noUpdate || delta === 0) return 0;
    if (delta < DURATION_MINUTE) return DURATION_SECOND;
    if (delta < DURATION_HOUR) return DURATION_MINUTE;
    if (delta < DURATION_DAY) return DURATION_HOUR;
    if (delta < DURATION_YEAR) return DURATION_DAY;
  }, [ delta, noUpdate ]);

  useEffect(() => {
    const timer = updateInterval
      ? setInterval(() => setNow(Date.now()), updateInterval)
      : undefined;

    return () => {
      if (timer) clearInterval(timer);
    };
  }, [ updateInterval ]);

  return (
    <Tooltip title={dayjs(milliseconds).format(TOOLTIP_FORMAT)}>
      <div className={classes.join(' ')}>{durationString}</div>
    </Tooltip>
  );
};

export default TimeAgo;
