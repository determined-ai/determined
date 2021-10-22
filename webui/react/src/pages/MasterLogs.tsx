import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import { V1MasterLogsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { jsonToMasterLogs } from 'services/decoder';
import { consumeStream } from 'services/utils';

const MasterLogs: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const logsRef = useRef<LogViewerHandles>(null);
  const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);

  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    if (startLogId >= oldestFetchedId) return;
    setOldestFetchedId(startLogId);
    consumeStream<V1MasterLogsResponse>(
      detApi.StreamingCluster.determinedMasterLogs(
        startLogId,
        oldestLogId-startLogId,
        false,
        { signal: canceler.signal },
      ),
      event => {
        const logEntry = (event as V1MasterLogsResponse).logEntry;
        if (logsRef.current && logEntry) {
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
  }, [ canceler.signal ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <LogViewer
      noWrap
      pageProps={{ title: 'Master Logs' }}
      ref={logsRef}
      onScrollToTop={handleScrollToTop}
    />
  );
};

export default MasterLogs;
