import React, { useCallback } from 'react';

import LogViewerCore, { FetchOptions, OffsetType } from 'components/LogViewerCore';
import { detApi } from 'services/apiConfig';
import { jsonToMasterLog } from 'services/decoder';

const MasterLogs: React.FC = () => {
  const handleFetch = useCallback((options: FetchOptions) => {
    return detApi.StreamingCluster.determinedMasterLogs(
      options.offset ?? 0,
      options.limit,
      options.follow,
      { signal: options.canceler.signal },
    );
  }, []);

  return (
    <LogViewerCore
      decoder={jsonToMasterLog}
      title="Master Logs"
      type={OffsetType.Id}
      onFetch={handleFetch}
    />
  );
};

export default MasterLogs;
