import React from 'react';

import PageMessage from 'components/PageMessage';

const ClusterNotAvailable: React.FC = () => {
  return (
    <PageMessage title="Cluster Not Available">
      <p>Cluster is not ready. Please try again later.</p>
    </PageMessage>
  );
};

export default ClusterNotAvailable;
