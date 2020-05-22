import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewer, { LogViewerHandles } from 'components/LogViewer';
import Section from 'components/Section';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getMasterLogs, LogsApiParams } from 'services/api';
import { Log } from 'types';

import css from './MasterLogs.module.scss';

const DEFAULT_PARAMS = { tail: 10000 };

const MasterLogs: React.FC = () => {
  const logsRef = useRef<LogViewerHandles>(null);
  const [ lastLogId, setLastLogId ] = useState(0);
  const [ logsResponse, setApiParams ] =
    useRestApiSimple<LogsApiParams, Log[]>(getMasterLogs, DEFAULT_PARAMS);

  const fetchLogs = useCallback(async (): Promise<void> => {
    if (!lastLogId) return;
    setApiParams({ greaterThanId: lastLogId });
  }, [ lastLogId, setApiParams ]);

  usePolling(fetchLogs);

  useEffect(() => {
    if (!logsResponse.data || logsResponse.data.length === 0) return;

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(logsResponse.data);

    // Update the last fetched last log to fetch newer entries next time.
    setLastLogId(logsResponse.data[logsResponse.data.length - 1].id);
  }, [ logsResponse.data ]);

  return (
    <div className={css.base}>
      <Section maxHeight title="Master Logs">
        <LogViewer fullPage ref={logsRef} />
      </Section>
    </div>
  );
};

export default MasterLogs;
