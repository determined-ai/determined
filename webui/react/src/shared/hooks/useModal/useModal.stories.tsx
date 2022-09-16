import { Button } from 'antd';
import React from 'react';

import useModal from './useModal';

export default {
  component: useModal,
  title: 'Shared/Modal',
};

const Container = () => {
  const { contextHolder, modalOpen } = useModal();

  return (
    <>
      <Button onClick={() => modalOpen()}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

export const Default = (): React.ReactNode => {
  return <Container />;
};
