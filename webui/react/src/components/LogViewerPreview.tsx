import { Dayjs } from 'dayjs';
import React, { PropsWithChildren, useCallback, useEffect, useRef, useState } from 'react';
import { debounce } from 'throttle-debounce';

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
  fetchLogs: (filters: LogViewerPreviewFilter, canceler: AbortController) => FetchArgs;
  fetchToLogConverter: (data: unknown) => TrialLog,
  onViewLogs?: () => void;
}

const DEBOUNCE_TIME = 1000;

const LogViewerPreview: React.FC<PropsWithChildren<Props>> = ({
  children,
  fetchLogs,
  fetchToLogConverter,
  onViewLogs,
}: PropsWithChildren<Props>) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [ logEntry, setLogEntry ] = useState<LogEntry>();

  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  const setupFetch = useCallback(() => {
    const filters = {};
    const canceler = new AbortController();
    let entry: TrialLog;

    const debounceFunc = debounce(DEBOUNCE_TIME, () => {
      setLogEntry({
        formattedTime: formatDatetime(entry.time, DATETIME_FORMAT),
        level: entry.level || LogLevel.Info,
        message: entry.message,
      });
    });

    consumeStream(
      fetchLogs(filters, canceler),
      event => {
        entry = fetchToLogConverter(event);
        debounceFunc();
      },
    );

    return { canceler, debounceFunc };
  }, [ fetchLogs, fetchToLogConverter ]);

  const handleClick = useCallback(() => {
    if (onViewLogs) onViewLogs();
  }, [ onViewLogs ]);

  useEffect(() => {
    const { canceler, debounceFunc } = setupFetch();
    return () => {
      canceler.abort();
      debounceFunc.cancel();
    };
  }, [ setupFetch ]);

  return (
    <div className={css.base}>
      {children}
      <div className={css.preview} onClick={handleClick}>
        <div className={css.frame}>
          <div className={css.container} ref={containerRef}>
            {logEntry && (
              <LogViewerEntry noWrap timeStyle={{ width: dateTimeWidth }} {...logEntry} />
            )}
          </div>
          <div className={css.icon}>
            <Icon name="expand" />
          </div>
        </div>
      </div>
    </div>
  );
};

export default LogViewerPreview;
