import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';

import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { setAuth } from 'stores/auth';
import { generateTestExperimentData } from 'storybook/shared/generateTestData';

const MODAL_TITLE = 'Fork';
const SHOW_FULL_CONFIG_TEXT = 'Show Full Config';

vi.mock('services/api', () => ({
  getResourcePools: () => Promise.resolve([]),
  getTaskTemplates: () => Promise.resolve([]),
  launchJupyterLab: () => Promise.resolve({ config: '' }),
}));

vi.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => <></>,
}));

const ModalTrigger: React.FC = () => {
  const ExperimentCreateModal = useModal(ExperimentCreateModalComponent);
  const { experiment, trial } = generateTestExperimentData();
  useEffect(() => {
    setAuth({ isAuthenticated: true });
  }, []);

  return (
    <>
      <Button onClick={ExperimentCreateModal.open}>Show Jupyter Lab</Button>
      <ExperimentCreateModal.Component
        {...{ experiment: experiment, trial: trial, type: CreateExperimentType.Fork }}
      />
    </>
  );
};

const Container: React.FC = () => <ModalTrigger />;

const setup = async () => {
  const user = userEvent.setup();

  render(<Container />);

  await user.click(screen.getByRole('button'));
};

describe('Create Experiment Modal', () => {
  it('modal can be opened', async () => {
    await setup();

    expect(await screen.findByText(MODAL_TITLE)).toBeInTheDocument();
  });

  it('modal defaults to simple config', async () => {
    await setup();

    expect(await screen.findByText(SHOW_FULL_CONFIG_TEXT)).toBeInTheDocument();
  });
});
