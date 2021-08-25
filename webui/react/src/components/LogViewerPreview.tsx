import { Dayjs } from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import { FetchArgs } from 'services/api-ts-sdk';
import { consumeStream } from 'services/utils';
import { LogLevel, TrialLog } from 'types';
import { formatDatetime } from 'utils/date';

import Icon from './Icon';
import LogViewerEntry, { DATETIME_FORMAT, LogEntry, MAX_DATETIME_LENGTH } from './LogViewerEntry';
import css from './LogViewerPreview.module.scss';

export interface LogViewerPreviewFilter {
  timestampAfter?: Dayjs,   // exclusive of the specified date time
}

interface Props {
  fetchToLogConverter: (data: unknown) => TrialLog,
  onFetchLogs: (filters: LogViewerPreviewFilter, canceler: AbortController) => FetchArgs;
  onViewLogs?: () => void;
}

const THROTTLE_TIME = 500;

const LogViewerPreview: React.FC<Props> = ({
  fetchToLogConverter,
  onFetchLogs,
  onViewLogs,
}: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [ logEntry, setLogEntry ] = useState<LogEntry>();

  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  const fetchLogs = useCallback(() => {
    const filters = {};
    const canceler = new AbortController();
    let entry: TrialLog;

    const throttleFunc = throttle(THROTTLE_TIME, () => {
      setLogEntry({
        formattedTime: formatDatetime(entry.time, DATETIME_FORMAT),
        level: entry.level || LogLevel.Info,
        message: entry.message,
      });
    });

    consumeStream(
      onFetchLogs(filters, canceler),
      event => {
        entry = fetchToLogConverter(event);
        throttleFunc();
      },
    );

    return { canceler, throttleFunc };
  }, [ fetchToLogConverter, onFetchLogs ]);

  const handleClick = useCallback(() => {
    if (onViewLogs) onViewLogs();
  }, [ onViewLogs ]);

  useEffect(() => {
    const { canceler, throttleFunc } = fetchLogs();
    return () => {
      canceler.abort();
      throttleFunc.cancel();
    };
  }, [ fetchLogs ]);

  return (
    <div className={css.base} onClick={handleClick}>
      <div className={css.frame}>
        <div className={css.container} ref={containerRef}>
          {logEntry && <LogViewerEntry noWrap timeStyle={{ width: dateTimeWidth }} {...logEntry} />}
        </div>
        <div className={css.icon}>
          <Icon name="expand" />
        </div>
      </div>
    </div>
  );
};

export default LogViewerPreview;
