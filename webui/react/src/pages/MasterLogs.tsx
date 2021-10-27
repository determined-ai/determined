import React, { useCallback, useEffect, useRef, useState } from 'react';

import LogViewerTimestamp, { TAIL_SIZE } from 'components/LogViewerTimestamp';
import { detApi } from 'services/apiConfig';
import { jsonToMasterLogs } from 'services/decoder';

const MasterLogs: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const oldestFetchedId = useRef<number>();
  const latestFetchedId = useRef<number>();

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const fetchLogAfter = useCallback(() => {
    return detApi.StreamingCluster.determinedMasterLogs(
      latestFetchedId.current ?? 0,
      TAIL_SIZE,
      false,
      { signal: canceler.signal },
    );
  }, [ canceler.signal ]);

  const fetchLogBefore = useCallback(() => {
    const offset = (oldestFetchedId.current ?? 0) - (latestFetchedId.current ?? 0) - TAIL_SIZE;
    return detApi.StreamingCluster.determinedMasterLogs(
      offset,
      TAIL_SIZE,
      false,
      { signal: canceler.signal },
    );
  }, [ canceler.signal ]);

  const fetchLogTail = useCallback(() => {
    return detApi.StreamingCluster.determinedMasterLogs(
      -TAIL_SIZE,
      0,
      true,
      { signal: canceler.signal },
    );
  }, [ canceler.signal ]);

  const fetchLogEndpoints = useCallback((oldest?: number, latest?: number) => {
    oldestFetchedId.current = oldest;
    latestFetchedId.current = latest;
  }, []);

  return (
    <LogViewerTimestamp
      fetchToLogConverter={jsonToMasterLogs}
      onFetchLogAfter={fetchLogAfter}
      onFetchLogBefore={fetchLogBefore}
      onFetchLogEndpoints={fetchLogEndpoints}
      onFetchLogTail={fetchLogTail} />
  );
};

export default MasterLogs;
