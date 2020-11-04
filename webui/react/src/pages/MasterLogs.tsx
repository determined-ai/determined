import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { getMasterLogs } from 'services/api';
import { LogsParams } from 'services/types';
import { Log } from 'types';

const MasterLogs: React.FC = () => {
  const logsRef = useRef<LogViewerHandles>(null);
  const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ logIdRange, setLogIdRange ] =
    useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  const [ logsResponse, triggerOldLogsRequest ] =
    useRestApi<LogsParams, Log[]>(getMasterLogs, { tail: TAIL_SIZE });
  const [ pollingLogsResponse, triggerNewLogsRequest ] =
    useRestApi<LogsParams, Log[]>(getMasterLogs, { tail: TAIL_SIZE });

  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    if (startLogId >= oldestFetchedId) return;
    setOldestFetchedId(startLogId);
    triggerOldLogsRequest({ greaterThanId: startLogId, tail: TAIL_SIZE });
  }, [ oldestFetchedId, triggerOldLogsRequest ]);

  const fetchNewerLogs = useCallback(() => {
    if (logIdRange.max < 0) return;
    triggerNewLogsRequest({ greaterThanId: logIdRange.max, tail: TAIL_SIZE });
  }, [ logIdRange.max, triggerNewLogsRequest ]);

  const handleScrollToTop = useCallback((oldestLogId: number) => {
    fetchOlderLogs(oldestLogId);
  }, [ fetchOlderLogs ]);

  usePolling(fetchNewerLogs);

  useEffect(() => {
    if (!logsResponse.data || logsResponse.data.length === 0) return;

    const minLogId = logsResponse.data[0].id;
    const maxLogId = logsResponse.data[logsResponse.data.length - 1].id;
    if (minLogId >= logIdRange.min) return;

    setLogIdRange({
      max: Math.max(logIdRange.max, maxLogId),
      min: Math.min(logIdRange.min, minLogId),
    });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(logsResponse.data, true);
  }, [ logIdRange, logsResponse ]);

  useEffect(() => {
    if (!pollingLogsResponse.data || pollingLogsResponse.data.length === 0) return;

    const minLogId = pollingLogsResponse.data[0].id;
    const maxLogId = pollingLogsResponse.data[pollingLogsResponse.data.length - 1].id;
    if (maxLogId <= logIdRange.max) return;

    setLogIdRange({
      max: Math.max(logIdRange.max, maxLogId),
      min: Math.min(logIdRange.min, minLogId),
    });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(pollingLogsResponse.data);
  }, [ logIdRange, pollingLogsResponse.data ]);

  return (
    <LogViewer
      noWrap
      ref={logsRef}
      title="Master Logs"
      onScrollToTop={handleScrollToTop} />
  );
};

export default MasterLogs;
