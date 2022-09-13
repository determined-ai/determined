import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewerEntry, {
  DATETIME_FORMAT, LogEntry, MAX_DATETIME_LENGTH,
} from 'components/LogViewer/LogViewerEntry';
import useGetCharMeasureInContainer from 'hooks/useGetCharMeasureInContainer';
import { detApi } from 'services/apiConfig';
import { mapV1LogsResponse } from 'services/decoder';
import { readStream } from 'services/utils';
import { formatDatetime } from 'shared/utils/datetime';
import { LogLevel, RunState, TrialDetails } from 'types';

import css from './TrialLogPreview.module.scss';

interface Props {
  children?: React.ReactNode;
  hidePreview?: boolean;
  onViewLogs?: () => void;
  trial?: TrialDetails;
}

const TrialLogPreview: React.FC<Props> = ({
  children,
  hidePreview = false,
  onViewLogs,
  trial,
}: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const nonEmptyLogFound = useRef(false);
  const [ logEntry, setLogEntry ] = useState<LogEntry>();

  const classes = [ css.base ];
  const charMeasures = useGetCharMeasureInContainer(containerRef);
  const dateTimeWidth = charMeasures.width * MAX_DATETIME_LENGTH;

  if (hidePreview || !logEntry) classes.push(css.hidePreview);

  const fetchTrialLogs = useCallback((trialId: number, time: string, canceler: AbortController) => {
    readStream(
      detApi.StreamingExperiments.trialLogs(
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
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        const entry = mapV1LogsResponse(event);
        setLogEntry({
          formattedTime: formatDatetime(entry.time, { format: DATETIME_FORMAT }),
          level: entry.level || LogLevel.Info,
          message: entry.message,
        });
      },
    );
  }, []);

  const fetchLatestTrialLog = useCallback((
    trialId: number,
    trialState: RunState,
    canceler: AbortController,
  ) => {
    readStream(
      detApi.StreamingExperiments.trialLogs(
        trialId,
        100,
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
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        const entry = mapV1LogsResponse(event);

        /*
         * Hoping within the 100 log lines we are able to find a log
         * entry that is not empty, so there is something we can show
         * as a baseline.
         */
        if (!nonEmptyLogFound.current && entry.message) {
          nonEmptyLogFound.current = true;
          fetchTrialLogs(trialId, entry.time, canceler);
        }
      },
    );
  }, [ fetchTrialLogs ]);

  const handleClick = useCallback(() => {
    if (onViewLogs) onViewLogs();
  }, [ onViewLogs ]);

  useEffect(() => {
    if (!trial?.id || trial?.state === RunState.Completed) return;

    const canceler = new AbortController();
    fetchLatestTrialLog(trial.id, trial.state, canceler);

    return () => canceler.abort();
  }, [ fetchLatestTrialLog, trial?.id, trial?.state ]);

  return (
    <div className={classes.join(' ')}>
      {children}
      <Tooltip mouseEnterDelay={0.25} title="View Logs">
        <div className={css.preview} onClick={handleClick}>
          <div className={css.container} ref={containerRef}>
            {logEntry && (
              <LogViewerEntry
                noWrap
                style={{ position: 'relative' }}
                timeStyle={{ width: dateTimeWidth }}
                {...logEntry}
              />
            )}
          </div>
        </div>
      </Tooltip>
    </div>
  );
};

export default TrialLogPreview;
