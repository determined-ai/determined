import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';

import LogViewer, { LogViewerHandles } from 'components/LogViewer';
import Page from 'components/Page';
import UI from 'contexts/UI';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { getTrialLogs } from 'services/api';
import { TrialLogsParams } from 'services/types';
import { Log } from 'types';
import { serverAddress } from 'utils/routes';

interface Params {
  trialId: string;
}

const TAIL_SIZE = 1000;

const TrialLogs: React.FC = () => {
  const { trialId } = useParams<Params>();
  const id = parseInt(trialId);
  const title = `Trial ${id} Logs`;
  const downloadUrl = `${serverAddress()}/trials/${id}/logs?format=raw`;
  const setUI = UI.useActionContext();
  const logsRef = useRef<LogViewerHandles>(null);
  const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ logIdRange, setLogIdRange ] =
    useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  const [ logsResponse, setLogsParams ] =
    useRestApiSimple<TrialLogsParams, Log[]>(getTrialLogs, { tail: TAIL_SIZE, trialId: id });
  const [ pollingLogsResponse, setPollingLogsParams ] =
    useRestApiSimple<TrialLogsParams, Log[]>(getTrialLogs, { tail: TAIL_SIZE, trialId: id });

  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    if (startLogId >= oldestFetchedId) return;
    setOldestFetchedId(startLogId);
    setLogsParams({ greaterThanId: startLogId, tail: TAIL_SIZE, trialId: id });
  }, [ id, oldestFetchedId, setLogsParams ]);

  const fetchNewerLogs = useCallback(() => {
    if (logIdRange.max < 0) return;
    setPollingLogsParams({ greaterThanId: logIdRange.max, tail: TAIL_SIZE, trialId: id });
  }, [ id, logIdRange.max, setPollingLogsParams ]);

  const handleScrollToTop = useCallback((oldestLogId: number) => {
    fetchOlderLogs(oldestLogId);
  }, [ fetchOlderLogs ]);

  usePolling(fetchNewerLogs);

  useEffect(() => {
    setUI({ type: UI.ActionType.HideChrome });
  }, [ setUI ]);

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
    <Page hideTitle maxHeight title={title}>
      <LogViewer
        disableLevel
        disableLineNumber
        downloadUrl={downloadUrl}
        isLoading={pollingLogsResponse.isLoading}
        noWrap
        ref={logsRef}
        title={title}
        onScrollToTop={handleScrollToTop} />
    </Page>
  );
};

export default TrialLogs;
