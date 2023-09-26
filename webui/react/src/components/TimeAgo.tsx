import dayjs, { Dayjs } from 'dayjs';
import React, { ReactNode, useEffect, useMemo, useState } from 'react';

import Tooltip from 'components/kit/Tooltip';
import { ValueOf } from 'types';
import { isNumber, isString } from 'utils/data';
import {
  DURATION_DAY,
  DURATION_HOUR,
  DURATION_MINUTE,
  DURATION_SECOND,
  DURATION_YEAR,
  durationInEnglish,
} from 'utils/datetime';
import { capitalize, capitalizeWord } from 'utils/string';

import css from './TimeAgo.module.scss';

export const TimeAgoCase = {
  Lower: 'lower',
  Sentence: 'sentence',
  Title: 'title',
} as const;

export type TimeAgoCase = ValueOf<typeof TimeAgoCase>;

interface Props {
  className?: string;
  dateFormat?: string;
  datetime: Dayjs | Date | number | string;
  long?: boolean;
  noUpdate?: boolean;
  stringCase?: TimeAgoCase;
  tooltipFormat?: string;
  units?: number;
}

export const JUST_NOW = 'Just Now';
export const DEFAULT_TOOLTIP_FORMAT = 'MMM D, YYYY - h:mm a';

const TimeAgo: React.FC<Props> = ({
  className,
  dateFormat = 'MMM D, YYYY',
  datetime,
  long = false,
  noUpdate = false,
  stringCase = TimeAgoCase.Sentence,
  tooltipFormat = DEFAULT_TOOLTIP_FORMAT,
  units = 1,
}: Props) => {
  const [now, setNow] = useState(() => Date.now());
  const classes: string[] = [css.base];

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
  }, [datetime]);
  const tooltip: ReactNode = useMemo(() => {
    let base = dayjs(milliseconds).format(tooltipFormat);
    const check = 'Last experiment started';

    if (base.includes(check)) {
      base = base.slice(check.length);

      // Tried to just add a line break \n to the string, but, it doesn't work...
      return (
        <>
          <span>{check}:</span>
          <br />
          <span>{base}</span>
        </>
      );
    }

    return base;
  }, [tooltipFormat, milliseconds]);

  const delta = useMemo(() => {
    return milliseconds === undefined ? 0 : now - milliseconds;
  }, [milliseconds, now]);

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
  }, [delta, dateFormat, long, milliseconds, units]);

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
  }, [duration, stringCase]);

  const updateInterval = useMemo(() => {
    if (noUpdate || delta === 0) return 0;
    if (delta < DURATION_MINUTE) return DURATION_SECOND;
    if (delta < DURATION_HOUR) return DURATION_MINUTE;
    if (delta < DURATION_DAY) return DURATION_HOUR;
    if (delta < DURATION_YEAR) return DURATION_DAY;
  }, [delta, noUpdate]);

  useEffect(() => {
    const timer = updateInterval
      ? setInterval(() => setNow(Date.now()), updateInterval)
      : undefined;

    return () => {
      if (timer) clearInterval(timer);
    };
  }, [updateInterval]);

  return (
    <Tooltip content={tooltip}>
      <div className={classes.join(' ')}>{durationString}</div>
    </Tooltip>
  );
};

export default TimeAgo;
