import React, { useEffect } from 'react';

import PageMessage from 'components/PageMessage';
import { StoreAction, useStoreDispatch } from 'contexts/Store';

const ClusterNotAvailable: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => storeDispatch({ type: StoreAction.HideUIChrome }), [ storeDispatch ]);

  return (
    <PageMessage title="Cluster Not Available">
      <p>Cluster is not ready. Please try again later.</p>
    </PageMessage>
  );
};

export default ClusterNotAvailable;
