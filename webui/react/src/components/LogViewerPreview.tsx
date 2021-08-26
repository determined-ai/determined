import { Tooltip } from 'antd';
import { Dayjs } from 'dayjs';
import React, { PropsWithChildren, useCallback, useEffect, useRef, useState } from 'react';
import { debounce } from 'throttle-debounce';

import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import { FetchArgs } from 'services/api-ts-sdk';
import { consumeStream } from 'services/utils';
import { LogLevel, TrialLog } from 'types';
import { formatDatetime } from 'utils/date';

import LogViewerEntry, { DATETIME_FORMAT, LogEntry, MAX_DATETIME_LENGTH } from './LogViewerEntry';
import css from './LogViewerPreview.module.scss';

export interface LogViewerPreviewFilter {
  timestampAfter?: Dayjs,   // exclusive of the specified date time
}

interface Props {
  fetchLogs: (filters: LogViewerPreviewFilter, canceler: AbortController) => FetchArgs;
  fetchToLogConverter: (data: unknown) => TrialLog,
  hidePreview?: boolean;
  onViewLogs?: () => void;
}

const DEBOUNCE_TIME = 100;

const LogViewerPreview: React.FC<PropsWithChildren<Props>> = ({
  children,
  fetchLogs,
  fetchToLogConverter,
  hidePreview = false,
  onViewLogs,
}: PropsWithChildren<Props>) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [ logEntry, setLogEntry ] = useState<LogEntry>();

  const classes = [ css.base ];
  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  if (hidePreview || !logEntry) classes.push(css.hidePreview);

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
    <div className={classes.join(' ')}>
      {children}
      <Tooltip mouseEnterDelay={0.25} title="View Logs">
        <div className={css.preview} onClick={handleClick}>
          <div className={css.container} ref={containerRef}>
            {logEntry && (
              <LogViewerEntry noWrap timeStyle={{ width: dateTimeWidth }} {...logEntry} />
            )}
          </div>
        </div>
      </Tooltip>
    </div>
  );
};

export default LogViewerPreview;
