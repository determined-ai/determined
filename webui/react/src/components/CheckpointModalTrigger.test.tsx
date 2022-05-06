import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

const TEST_MODAL_TITLE = 'Checkpoint Modal Test';
const REGISTER_CHECKPOINT_TEXT = 'Register Checkpoint';

jest.mock('services/api', () => ({
  getModels: () => {
    return Promise.resolve({ models: [] });
  },
}));

const ModalTrigger: React.FC = () => {
  const { experiment, checkpoint } = generateTestExperimentData();

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <CheckpointModalTrigger
      checkpoint={checkpoint}
      experiment={experiment}
      title={TEST_MODAL_TITLE}
    />
  );
};

const ModalTriggerContainer: React.FC = () => {
  return (
    <StoreProvider>
      <ModalTrigger />
    </StoreProvider>
  );
};

const setup = async () => {

  render(
    <ModalTriggerContainer />,
  );
  userEvent.click(await screen.findByRole('button'));
};

describe('CheckpointModalTrigger', () => {
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
