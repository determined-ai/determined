import React from 'react';

import ResourcePoolCard from './ResourcePoolCard';

export default {
  component: ResourcePoolCard,
  title: 'ResourcePoolCard',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolCard containerStates={[]} rpIndex={0} />;
};
