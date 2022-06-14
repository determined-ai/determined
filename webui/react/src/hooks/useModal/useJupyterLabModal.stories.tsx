import { Button, Modal } from 'antd';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import useJupyterLabModal from 'hooks/useModal/useJupyterLabModal';
import StoreDecorator from 'storybook/StoreDecorator';

export default {
  component: useJupyterLabModal,
  decorators: [ StoreDecorator ],
  title: 'useJupyterLabModal',
};

const UseJupyterLabModalContainer = () => {
  const storeDispatch = useStoreDispatch();
  const [ jupyterLabModal, jupyterLabModalContextHolder ] = Modal.useModal();
  const { modalOpen } = useJupyterLabModal(jupyterLabModal);
  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <>
      <Button onClick={() => modalOpen()}>Open Jupyter Lab</Button>
      {jupyterLabModalContextHolder}
    </>
  );
};

export const Default = (): React.ReactNode => {
  return <UseJupyterLabModalContainer />;
};
