import { Button } from 'antd';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';
import StoreDecorator from 'storybook/StoreDecorator';

import useModalCheckpoint from './useModalCheckpoint';

export default {
  component: useModalCheckpoint,
  decorators: [ StoreDecorator ],
  title: 'useModalCheckpoint',
};

const UseCheckpointModalContainer = () => {

  const { checkpoint, experiment } = generateTestExperimentData();

  const { modalOpen } = useModalCheckpoint({
    checkpoint: checkpoint,
    config: experiment.config,
    title: 'Use Checkpoint Modal',
  });

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <Button onClick={() => modalOpen()}>View Checkpoint</Button>;
};

export const Default = (): React.ReactNode => {
  return <UseCheckpointModalContainer />;
};
