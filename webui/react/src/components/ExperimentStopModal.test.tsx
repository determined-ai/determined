import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import ExperimentStopModalComponent, {
  BUTTON_TEXT,
  CHECKBOX_TEXT,
} from 'components/ExperimentStopModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import {
  cancelExperiment as mockCancelExperiment,
  killExperiment as mockKillExperiment,
} from 'services/api';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

const user = userEvent.setup();

vi.mock('services/api', () => ({
  cancelExperiment: vi.fn(),
  killExperiment: vi.fn(),
}));

const { experiment } = generateTestExperimentData();

const ModalTrigger: React.FC = () => {
  const ExperimentStopModal = useModal(ExperimentStopModalComponent);

  return (
    <>
      <Button onClick={ExperimentStopModal.open} />
      <ExperimentStopModal.Component experimentId={experiment.id} />
    </>
  );
};

const setup = async () => {
  render(<ModalTrigger />);

  await user.click(screen.getByRole('button'));
};

describe('Stop Experiment Modal', () => {
  it('submits a valid cancel experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: BUTTON_TEXT }));

    expect(mockCancelExperiment).toHaveBeenCalledWith({ experimentId: experiment.id });
  });

  it('submits a valid kill experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('checkbox', { name: CHECKBOX_TEXT }));
    await user.click(screen.getByRole('button', { name: BUTTON_TEXT }));

    expect(mockKillExperiment).toHaveBeenCalledWith({ experimentId: experiment.id });
  });
});
