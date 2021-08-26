import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import React, { PropsWithChildren, useCallback, useEffect, useRef, useState } from 'react';

import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import { detApi } from 'services/apiConfig';
import { jsonToTrialLog } from 'services/decoder';
import { consumeStream } from 'services/utils';
import { LogLevel } from 'types';
import { formatDatetime } from 'utils/date';

import LogViewerEntry, { DATETIME_FORMAT, LogEntry, MAX_DATETIME_LENGTH } from './LogViewerEntry';
import css from './TrialLogPreview.module.scss';

interface Props {
  hidePreview?: boolean;
  onViewLogs?: () => void;
  trialId?: number;
}

const TrialLogPreview: React.FC<PropsWithChildren<Props>> = ({
  children,
  hidePreview = false,
  onViewLogs,
  trialId,
}: PropsWithChildren<Props>) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [ logEntry, setLogEntry ] = useState<LogEntry>();

  const classes = [ css.base ];
  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  if (hidePreview || !logEntry) classes.push(css.hidePreview);

  const fetchTrialLogs = useCallback((trialId: number, time: string, canceler: AbortController) => {
    consumeStream(
      detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        undefined,
        true,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        dayjs(time).toDate(),
        'ORDER_BY_ASC',
        { signal: canceler.signal },
      ),
      event => {
        const entry = jsonToTrialLog(event);
        setLogEntry({
          formattedTime: formatDatetime(entry.time, DATETIME_FORMAT),
          level: entry.level || LogLevel.Info,
          message: entry.message,
        });
      },
    );
  }, []);

  const fetchLatestTrialLog = useCallback((trialId: number, canceler: AbortController) => {
    consumeStream(
      detApi.StreamingExperiments.determinedTrialLogs(
        trialId,
        1,
        false,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        'ORDER_BY_DESC',
        { signal: canceler.signal },
      ),
      event => {
        const entry = jsonToTrialLog(event);
        fetchTrialLogs(trialId, entry.time, canceler);
      },
    );
  }, [ fetchTrialLogs ]);

  const handleClick = useCallback(() => {
    if (onViewLogs) onViewLogs();
  }, [ onViewLogs ]);

  useEffect(() => {
    if (!trialId) return;

    const canceler = new AbortController();
    fetchLatestTrialLog(trialId, canceler);
    return () => canceler.abort();
  }, [ fetchLatestTrialLog, trialId ]);

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

export default TrialLogPreview;
