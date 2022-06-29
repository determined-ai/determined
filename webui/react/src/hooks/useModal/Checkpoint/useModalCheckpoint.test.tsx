import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

import useModalCheckpoint from './useModalCheckpoint';

const TEST_MODAL_TITLE = 'Checkpoint Modal Test';
const MODAL_TRIGGER_TEXT = 'Open Checkpoint Modal';
const REGISTER_CHECKPOINT_TEXT = 'Register Checkpoint';

jest.mock('services/api', () => ({
  getModels: () => {
    return Promise.resolve({ models: [] });
  },
}));

const ModalTriggerButton: React.FC = () => {
  const { experiment, checkpoint } = generateTestExperimentData();
  const storeDispatch = useStoreDispatch();

  const { contextHolder, modalOpen } = useModalCheckpoint({
    checkpoint: checkpoint,
    config: experiment.config,
    title: TEST_MODAL_TITLE,
  });

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <>
      <Button onClick={() => modalOpen()}>{MODAL_TRIGGER_TEXT}</Button>
      {contextHolder}
    </>
  );
};

const Container: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTriggerButton />
    </StoreProvider>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(<Container />);

  await user.click(screen.getByText(MODAL_TRIGGER_TEXT));

  return user;
};

describe('useModalCheckpoint', () => {
  it('should open modal', async () => {
    await setup();

    expect(await screen.findByText(TEST_MODAL_TITLE)).toBeInTheDocument();
  });

  it('should close modal', async () => {
    const user = await setup();

    await screen.findByText(TEST_MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    await waitFor(() => {
      expect(screen.queryByText(TEST_MODAL_TITLE)).not.toBeInTheDocument();
    });
  });

  it('open register checkpoint modal', async () => {
    const user = await setup();

    await screen.findByText(TEST_MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: REGISTER_CHECKPOINT_TEXT }));

    await waitFor(() => {
      screen.debug();
      expect(screen.queryByText(REGISTER_CHECKPOINT_TEXT)).toBeInTheDocument();
    });
  });
});
