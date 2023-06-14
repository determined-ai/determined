import React, { useCallback } from 'react';

import LogViewer, {
  FetchConfig,
  FetchDirection,
  FetchType,
} from 'components/kit/LogViewer/LogViewer';
import { serverAddress } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { jsonToClusterLog } from 'services/decoder';
import { isNumber } from 'utils/data';
import handleError from 'utils/error';

import css from './ClusterLogs.module.scss';

const ClusterLogs: React.FC = () => {
  const handleFetch = useCallback((config: FetchConfig, type: FetchType) => {
    const options = { follow: false, limit: config.limit, offset: 0 };
    const offsetId = isNumber(config.offsetLog?.id) ? config.offsetLog?.id ?? 0 : 0;

    if (type === FetchType.Initial) {
      if (config.fetchDirection === FetchDirection.Older) options.offset = -config.limit;
    } else if (type === FetchType.Newer) {
      options.offset = offsetId;
    } else if (type === FetchType.Older) {
      options.offset = Math.max(0, offsetId - config.limit);
    } else if (type === FetchType.Stream) {
      options.offset = -1;
      options.follow = true;
      options.limit = 0;
    }

    return detApi.StreamingCluster.masterLogs(options.offset, options.limit, options.follow, {
      signal: config.canceler.signal,
    });
  }, []);

  return (
    <div className={css.base}>
      <LogViewer
        decoder={jsonToClusterLog}
        serverAddress={serverAddress}
        sortKey="id"
        onError={handleError}
        onFetch={handleFetch}
      />
    </div>
  );
};

export default ClusterLogs;
