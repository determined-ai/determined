import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import StoreDecorator from 'storybook/StoreDecorator';

import JupyterLabModal from './JupyterLabModal';

export default {
  component: JupyterLabModal,
  decorators: [ StoreDecorator ],
  title: 'JupyterLabModal',
};

const JupyterLabModalContainer = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <JupyterLabModal visible={true} />;
};

export const Default = (): React.ReactNode => {
  return <JupyterLabModalContainer />;
};
