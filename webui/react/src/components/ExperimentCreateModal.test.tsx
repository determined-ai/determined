import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useInitApi } from 'hew/Toast';
import React from 'react';

import ExperimentCreateModalComponent, {
  CreateExperimentType,
  FULL_CONFIG_BUTTON_TEXT,
  SIMPLE_CONFIG_BUTTON_TEXT,
} from 'components/ExperimentCreateModal';
import { ThemeProvider } from 'components/ThemeProvider';
import { createExperiment as mockCreateExperiment } from 'services/api';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

const user = userEvent.setup();

vi.mock('services/api', () => ({
  createExperiment: vi.fn(),
}));

const ModalTrigger: React.FC = () => {
  const ExperimentCreateModal = useModal(ExperimentCreateModalComponent);
  const { experiment, trial } = generateTestExperimentData();
  useInitApi();
  return (
    <>
      <Button onClick={ExperimentCreateModal.open} />
      <ExperimentCreateModal.Component
        experiment={experiment}
        trial={trial}
        type={CreateExperimentType.Fork}
      />
    </>
  );
};

const setup = async () => {
  render(
    <App>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <ModalTrigger />
        </ThemeProvider>
      </UIProvider>
    </App>,
  );

  await user.click(screen.getByRole('button'));
};

describe('Create Experiment Modal', () => {
  it('defaults to simple config', async () => {
    await setup();

    expect(await screen.findByText(FULL_CONFIG_BUTTON_TEXT)).toBeInTheDocument();
  });

  it('changes to full config', async () => {
    await setup();

    await user.click(screen.getByText(FULL_CONFIG_BUTTON_TEXT));

    expect(await screen.findByText(SIMPLE_CONFIG_BUTTON_TEXT)).toBeInTheDocument();
  });

  it('submits a valid create experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: CreateExperimentType.Fork }));
    expect(mockCreateExperiment).toHaveBeenCalled();
  });
});
