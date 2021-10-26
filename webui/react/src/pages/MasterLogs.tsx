import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import LogViewerTimestamp from 'components/LogViewerTimestamp';
import { V1MasterLogsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { jsonToMasterLogs, jsonToTrialLog } from 'services/decoder';
import { consumeStream } from 'services/utils';

const MasterLogs: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const logsRef = useRef<LogViewerHandles>(null);
  const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ latestFetchedId, setLatestFetchedId ] = useState(0);
  /*
  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    if (startLogId >= oldestFetchedId) return;
    setOldestFetchedId(startLogId);
    consumeStream<V1MasterLogsResponse>(
      detApi.StreamingCluster.determinedMasterLogs(
        startLogId,
        Math.max(0, oldestLogId-startLogId),
        false,
        { signal: canceler.signal },
      ),
      event => {
        const logEntry = (event as V1MasterLogsResponse).logEntry;
        if (logEntry) {
          logsRef.current?.addLogs(jsonToMasterLogs(logEntry), true);
        }
      },
    );
  }, [ oldestFetchedId, canceler.signal ]);

  const handleScrollToTop = useCallback((oldestLogId: number) => {
    fetchOlderLogs(oldestLogId);
  }, [ fetchOlderLogs ]);

  useEffect(() => {
    consumeStream<V1MasterLogsResponse>(
      detApi.StreamingCluster.determinedMasterLogs(
        -TAIL_SIZE,
        0,
        true,
        { signal: canceler.signal },
      ),
      event => {
        const logEntry = (event as V1MasterLogsResponse).logEntry;
        if (logsRef.current && logEntry) {
          logsRef.current?.addLogs(jsonToMasterLogs(logEntry));
        }
      },
    );
  }, [ canceler.signal ]); */

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const fetchLogAfter = useCallback(() => {
    return detApi.StreamingCluster.determinedMasterLogs(
      latestFetchedId,
      TAIL_SIZE,
      false,
      { signal: canceler.signal },
    );
  }, [ canceler.signal, latestFetchedId ]);

  const fetchLogBefore = useCallback(() => {
    /* const startLogId = Math.max(0, oldestFetchedId - TAIL_SIZE);
    setOldestFetchedId(startLogId);
    return detApi.StreamingCluster.determinedMasterLogs(
      startLogId,
      TAIL_SIZE,
      false,
      { signal: canceler.signal },
    ); */
    return detApi.StreamingCluster.determinedMasterLogs(
      latestFetchedId,
      TAIL_SIZE,
      false,
      { signal: canceler.signal },
    );
  }, [ canceler.signal, latestFetchedId ]);

  const fetchLogTail = useCallback(() => {
    return detApi.StreamingCluster.determinedMasterLogs(
      -TAIL_SIZE,
      0,
      true,
      { signal: canceler.signal },
    );
  }, [ canceler.signal ]);

  return (

    <LogViewerTimestamp
      fetchToLogConverter={jsonToMasterLogs}
      onFetchLogAfter={fetchLogAfter}
      onFetchLogBefore={fetchLogBefore}
      onFetchLogTail={fetchLogTail} />

  );
};

export default MasterLogs;
