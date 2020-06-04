import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewer, { LogViewerHandles } from 'components/LogViewer';
import Navigation from 'contexts/Navigation';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getMasterLogs } from 'services/api';
import { LogsParams } from 'services/types';
import { Log } from 'types';

import css from './MasterLogs.module.scss';

const TAIL_SIZE = 10000;

const MasterLogs: React.FC = () => {
  const setNavigation = Navigation.useActionContext();
  const logsRef = useRef<LogViewerHandles>(null);
  const [ logIdRange, setLogIdRange ] =
    useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  const [ logsResponse, setLogsParams ] =
    useRestApiSimple<LogsParams, Log[]>(getMasterLogs, { tail: TAIL_SIZE });
  const [ pollingLogsResponse, setPollingLogsParams ] =
    useRestApiSimple<LogsParams, Log[]>(getMasterLogs, { tail: TAIL_SIZE });

  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    if (logsResponse.isLoading) return;
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    setLogsParams({ greaterThanId: startLogId, tail: TAIL_SIZE });
  }, [ logsResponse, setLogsParams ]);

  const fetchNewerLogs = useCallback(() => {
    if (!logIdRange.max || pollingLogsResponse.isLoading) return;
    setPollingLogsParams({ greaterThanId: logIdRange.max, tail: TAIL_SIZE });
  }, [ logIdRange, pollingLogsResponse, setPollingLogsParams ]);

  const handleLoadOlderLogs = useCallback((oldestLogId: number) => {
    /*
     * Check to see if already at the oldest log where log id is 0
     * or if we already have the older logs we are trying to fetch.
     */
    if (oldestLogId === 0 || oldestLogId >= logIdRange.min) return;

    fetchOlderLogs(oldestLogId);
  }, [ fetchOlderLogs, logIdRange ]);

  usePolling(fetchNewerLogs);

  useEffect(() => {
    setNavigation({ type: Navigation.ActionType.Set, value: { showChrome: false } });
  }, [ setNavigation ]);

  useEffect(() => {
    if (!logsResponse.data || logsResponse.data.length === 0) return;

    const minLogId = logsResponse.data[0].id;
    if (minLogId >= logIdRange.min) return;

    setLogIdRange({ max: logIdRange.max, min: Math.min(logIdRange.min, minLogId) });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(logsResponse.data, true);
  }, [ logIdRange, logsResponse ]);

  useEffect(() => {
    if (!pollingLogsResponse.data || pollingLogsResponse.data.length === 0) return;

    const maxLogId = pollingLogsResponse.data[pollingLogsResponse.data.length - 1].id;
    if (maxLogId <= logIdRange.max) return;

    setLogIdRange({ max: Math.max(logIdRange.max, maxLogId), min: logIdRange.min });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(pollingLogsResponse.data);
  }, [ logIdRange, pollingLogsResponse.data ]);

  return (
    <div className={css.base}>
      <LogViewer ref={logsRef} title="Master Logs" onLoadOlderLogs={handleLoadOlderLogs} />
    </div>
  );
};

export default MasterLogs;
