import React, { useCallback } from 'react';

import LogViewerCore, { FetchConfig, FetchType } from 'components/LogViewerCore';
import Page from 'components/Page';
import { detApi } from 'services/apiConfig';
import { jsonToMasterLog } from 'services/decoder';

import css from './MasterLogs.module.scss';

const MasterLogs: React.FC = () => {
  const handleFetch = useCallback((config: FetchConfig, type: FetchType) => {
    const options = { follow: false, limit: config.limit, offset: 0 };

    if (type === FetchType.Initial) {
      if (config.isNewestFirst) options.offset = -config.limit;
    } else if (type === FetchType.Newer) {
      options.offset = config.offsetLog?.id ?? 0;
    } else if (type === FetchType.Older) {
      options.offset = Math.max(0, (config.offsetLog?.id ?? 0) - config.limit);
    } else if (type === FetchType.Stream) {
      options.offset = -1;
      options.follow = true;
      options.limit = 0;
    }

    return detApi.StreamingCluster.masterLogs(
      options.offset,
      options.limit,
      options.follow,
      { signal: config.canceler.signal },
    );
  }, []);

  return (
    <Page bodyNoPadding id="master-logs">
      <LogViewerCore
        decoder={jsonToMasterLog}
        sortKey="id"
        title={<div className={css.title}>Master Logs</div>}
        onFetch={handleFetch}
      />
    </Page>
  );
};

export default MasterLogs;
