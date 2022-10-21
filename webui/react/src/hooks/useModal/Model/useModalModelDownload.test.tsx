import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestModelVersion } from 'storybook/shared/generateTestData';

import useModalModelDownload from './useModalModelDownload';

const modelVersion = generateTestModelVersion();

const ModalTrigger: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { contextHolder, modalOpen } = useModalModelDownload();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return (
    <>
      <Button onClick={() => modalOpen(modelVersion)}>Download</Button>
      {contextHolder}
    </>
  );
};

const Container: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTrigger />
    </StoreProvider>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(<Container />);

  await user.click(screen.getByRole('button'));
};

describe('useModalExperimentCreate', () => {
  it('modal can be opened', async () => {
    await setup();

    expect(await screen.findByText('Download Model Command')).toBeInTheDocument();
  });

  it('modal contains checkpoint id', async () => {
    await setup();

    expect(await screen.getByDisplayValue(modelVersion.checkpoint.uuid)).toBeInTheDocument();
  });
});
