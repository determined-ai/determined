import { Button } from 'antd';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

import useModalCheckpoint from './useModalCheckpoint';

export default {
  component: useModalCheckpoint,
  title: 'useModalCheckpoint',
};

const Container = () => {
  const storeDispatch = useStoreDispatch();
  const { checkpoint, experiment } = generateTestExperimentData();

  const { modalOpen } = useModalCheckpoint({
    checkpoint: checkpoint,
    config: experiment.config,
    title: 'Use Checkpoint Modal',
  });

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return <Button onClick={() => modalOpen()}>View Checkpoint</Button>;
};

export const Default = (): React.ReactNode => <Container />;
