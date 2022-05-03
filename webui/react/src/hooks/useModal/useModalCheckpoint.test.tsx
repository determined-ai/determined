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

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  const { modalOpen } = useModalCheckpoint({
    checkpoint: checkpoint,
    config: experiment.config,
    title: TEST_MODAL_TITLE,
  });

  return (
    <Button onClick={() => modalOpen()}>{MODAL_TRIGGER_TEXT}</Button>
  );
};

const ModalTriggerContainer: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTriggerButton />
    </StoreProvider>
  );
};

const setup = async () => {

  render(
    <ModalTriggerContainer />,
  );
  userEvent.click(await screen.findByText(MODAL_TRIGGER_TEXT));
};

describe('useModalCheckpoint', () => {
  it('open modal', async () => {
    await setup();

    expect(await screen.findByText(TEST_MODAL_TITLE)).toBeInTheDocument();
  });

  it('close modal', async () => {
    await setup();

    await screen.findByText(TEST_MODAL_TITLE);

    userEvent.click(screen.getByRole('button', { name: /cancel/i }));

    await waitFor(() => {
      expect(screen.queryByText(TEST_MODAL_TITLE)).not.toBeInTheDocument();
    });
  });

  it('open register checkpoint modal', async () => {
    await setup();

    await screen.findByText(TEST_MODAL_TITLE);

    userEvent.click(screen.getByRole('button', { name: /Register Checkpoint/i }));

    await waitFor(() => {
      expect(screen.queryByText(REGISTER_CHECKPOINT_TEXT)).toBeInTheDocument();
    });
  });

});
