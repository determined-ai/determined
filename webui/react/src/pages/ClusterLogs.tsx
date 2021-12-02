import React, { useCallback } from 'react';

import LogViewer, { FetchConfig, FetchType } from 'components/LogViewer/LogViewer';
import Page from 'components/Page';
import { detApi } from 'services/apiConfig';
import { jsonToClusterLog } from 'services/decoder';
import { isNumber } from 'utils/data';

import css from './ClusterLogs.module.scss';

const ClusterLogs: React.FC = () => {
  const handleFetch = useCallback((config: FetchConfig, type: FetchType) => {
    const options = { follow: false, limit: config.limit, offset: 0 };
    const offsetId = isNumber(config.offsetLog?.id) ? config.offsetLog?.id ?? 0 : 0;

    if (type === FetchType.Initial) {
      if (config.isNewestFirst) options.offset = -config.limit;
    } else if (type === FetchType.Newer) {
      options.offset = offsetId;
    } else if (type === FetchType.Older) {
      options.offset = Math.max(0, offsetId - config.limit);
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
      <LogViewer
        decoder={jsonToClusterLog}
        sortKey="id"
        title={<div className={css.title}>Cluster Logs</div>}
        onFetch={handleFetch}
      />
    </Page>
  );
};

export default ClusterLogs;
