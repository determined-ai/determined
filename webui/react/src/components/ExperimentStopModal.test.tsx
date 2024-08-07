import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import React from 'react';

import ExperimentStopModalComponent, { CHECKBOX_TEXT } from 'components/ExperimentStopModal';
import { ThemeProvider } from 'components/ThemeProvider';
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

const mockUseFeature = vi.hoisted(() => vi.fn(() => false));
vi.mock('hooks/useFeature', () => {
  return {
    default: () => ({
      isOn: mockUseFeature,
    }),
  };
});

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
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ModalTrigger />
      <ThemeProvider />
    </UIProvider>,
  );

  await user.click(screen.getByRole('button'));
};

describe('Stop Experiment Modal', () => {
  afterEach(() => {
    mockUseFeature.mockClear();
  });
  it('submits a valid cancel experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: /Stop/ }));

    expect(mockCancelExperiment).toHaveBeenCalledWith({ experimentId: experiment.id });
  });

  it('submits a valid kill experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('checkbox', { name: CHECKBOX_TEXT }));
    await user.click(screen.getByRole('button', { name: /Stop/ }));

    expect(mockKillExperiment).toHaveBeenCalledWith({ experimentId: experiment.id });
  });
});
