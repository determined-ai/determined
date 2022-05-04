import React, { useEffect } from 'react';
import {Button} from 'antd';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import StoreDecorator from 'storybook/StoreDecorator';

import useJupyterLabModal from 'hooks/useModal/useJupyterLabModal';

export default {
  component: useJupyterLabModal,
  decorators: [ StoreDecorator ],
  title: 'useJupyterLabModal',
};

const UseJupyterLabModalContainer = () => {
  const storeDispatch = useStoreDispatch();
  const {modalOpen} = useJupyterLabModal();
  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <Button onClick={() => modalOpen()}> Open Jupyter Lab</Button>;
};

export const Default = (): React.ReactNode => {
  return <UseJupyterLabModalContainer/>;
};
