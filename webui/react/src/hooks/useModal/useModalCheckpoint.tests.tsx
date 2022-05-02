import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React from 'react';

import { generateTestExperimentData } from 'storybook/shared/generateTestExperiments';
import useModalCheckpoint from './useModalCheckpoint';

const TEST_MODAL_TITLE = 'Checkpoint Modal Test';
const MODAL_TRIGGER_TEXT = 'Open Checkpoint Modal'

const ModalTriggerButton: React.FC = () => {
  const {experiment, checkpoint} = generateTestExperimentData();

  const { modalOpen } = useModalCheckpoint({
    config: experiment.config,
    checkpoint: checkpoint,
    title: TEST_MODAL_TITLE 
  });

  return (
    <Button onClick={() => modalOpen()}>{MODAL_TRIGGER_TEXT}</Button>
  );
};

const setup = async () => {

  render(
    < ModalTriggerButton/>,
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

    await screen.findByText(TEST_MODAL_TITLE );

    userEvent.click(screen.getByRole('button', { name: /cancel/i }));

    await waitFor(() => {
      expect(screen.queryByText(TEST_MODAL_TITLE )).not.toBeInTheDocument();
    });
  });

});
