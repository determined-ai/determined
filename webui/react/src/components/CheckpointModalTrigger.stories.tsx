import React, { useEffect } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

export default {
  component: CheckpointModalTrigger,
  title: 'Determined/CheckpointModalTrigger',
};

const CheckpointModalTriggerContainer = () => {
  const storeDispatch = useStoreDispatch();
  const { checkpoint, experiment } = generateTestExperimentData();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return (
    <CheckpointModalTrigger
      checkpoint={checkpoint}
      experiment={experiment}
      title="CheckpointModalTrigger"
    />
  );
};

export const Default = (): React.ReactNode => {
  return <CheckpointModalTriggerContainer />;
};
