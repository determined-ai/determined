import { Button, Modal } from 'antd';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';
import StoreDecorator from 'storybook/StoreDecorator';

export default {
  component: useModalJupyterLab,
  decorators: [ StoreDecorator ],
  title: 'useModalJupyterLab',
};

const Container = () => {
  const storeDispatch = useStoreDispatch();
  const [ jupyterLabModal, jupyterLabModalContextHolder ] = Modal.useModal();
  const { modalOpen } = useModalJupyterLab(jupyterLabModal);

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
  return <Container />;
};
