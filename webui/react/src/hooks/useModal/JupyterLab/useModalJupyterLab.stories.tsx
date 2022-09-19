import { Button } from 'antd';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';

export default {
  component: useModalJupyterLab,
  title: 'useModalJupyterLab',
};

const Container = () => {
  const storeDispatch = useStoreDispatch();
  const { contextHolder, modalOpen } = useModalJupyterLab();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return (
    <>
      <Button onClick={() => modalOpen()}>Open Jupyter Lab</Button>
      {contextHolder}
    </>
  );
};

export const Default = (): React.ReactNode => {
  return <Container />;
};
