import { Button } from 'antd';
import React from 'react';

import StoreDecorator from 'storybook/StoreDecorator';

import useModalCheckpoint from './useModalCheckpoint';

export default {
  component: useModalCheckpoint,
  decorators: [ StoreDecorator ],
  title: 'CheckpointModal',
};

export const Default = (): React.ReactNode => {

  return (
    <Button disabled>View Checkpoint</Button>
  );
};
