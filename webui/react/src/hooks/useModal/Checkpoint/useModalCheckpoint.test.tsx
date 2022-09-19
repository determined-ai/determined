import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useCallback } from 'react';

import StoreProvider from 'contexts/Store';
import { ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';

import useModalCheckpoint, { Props } from './useModalCheckpoint';

const TEST_MODAL_TITLE = 'Checkpoint Modal Test';
const MODAL_TRIGGER_TEXT = 'Open Checkpoint Modal';
const REGISTER_CHECKPOINT_TEXT = 'Register Checkpoint';

jest.mock('services/api', () => ({
  getModels: () => {
    return Promise.resolve({ models: [] });
  },
}));

const { experiment, checkpoint } = generateTestExperimentData();

const Container: React.FC = (props: Partial<Props> = {}) => {
  const { contextHolder, modalOpen } = useModalCheckpoint({
    checkpoint: checkpoint,
    config: experiment.config,
    title: TEST_MODAL_TITLE,
    ...props,
  });

  const handleClick = useCallback(() => modalOpen(), [ modalOpen ]);

  return (
    <StoreProvider>
      <Button onClick={handleClick}>{MODAL_TRIGGER_TEXT}</Button>
      {contextHolder}
    </StoreProvider>
  );
};

const setup = async (props: Partial<Props> = {}) => {
  const user = userEvent.setup();

  render(<Container {...props} />);

  await user.click(screen.getByText(MODAL_TRIGGER_TEXT));

  return user;
};

describe('useModalCheckpoint', () => {
  it('should open modal', async () => {
    await setup();

    expect(await screen.findByText(TEST_MODAL_TITLE)).toBeInTheDocument();
  });

  it('should close modal', async () => {
    const onClose = jest.fn();
    const user = await setup({ onClose });

    await screen.findByText(TEST_MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    expect(onClose).toHaveBeenCalledWith(ModalCloseReason.Cancel);

    await waitFor(() => {
      expect(screen.queryByText(TEST_MODAL_TITLE)).not.toBeInTheDocument();
    });
  });

  it('should call `onClose` handler with Okay', async () => {
    const onClose = jest.fn();
    const user = await setup({ onClose });

    await screen.findByText(TEST_MODAL_TITLE);

    await user.click(screen.getByRole('button', { name: REGISTER_CHECKPOINT_TEXT }));

    expect(onClose).toHaveBeenCalledWith(ModalCloseReason.Ok);
  });
});
