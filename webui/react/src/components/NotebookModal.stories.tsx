import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import StoreDecorator from 'storybook/StoreDecorator';

import NotebookModal from './NotebookModal';

export default {
  component: NotebookModal,
  decorators: [ StoreDecorator ],
  title: 'NotebookModal',
};

const NotebookModalContainer = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <NotebookModal visible={true} />;
};

export const Default = (): React.ReactNode => {
  return <NotebookModalContainer />;
};
