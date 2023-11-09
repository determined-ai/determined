import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import React from 'react';

import ExperimentDeleteModalComponent, { BUTTON_TEXT } from 'components/ExperimentDeleteModal';
import { deleteExperiment as mockDeleteExperiment } from 'services/api';
import { generateTestExperimentData } from 'utils/tests/generateTestData';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ThemeProvider } from './ThemeProvider';
import Default from 'pages/DefaultRoute';

const user = userEvent.setup();

vi.mock('services/api', () => ({
  deleteExperiment: vi.fn(),
}));

vi.mock('utils/routes', () => ({
  routeToReactUrl: vi.fn(),
}));

const { experiment } = generateTestExperimentData();

const ModalTrigger: React.FC = () => {
  const ExperimentDeleteModal = useModal(ExperimentDeleteModalComponent);

  return (
    <>
      <Button onClick={ExperimentDeleteModal.open} />
      <ExperimentDeleteModal.Component experiment={experiment} />
    </>
  );
};

const setup = async () => {
  render(<UIProvider theme={DefaultTheme.Light}><ModalTrigger /><ThemeProvider></ThemeProvider></UIProvider>);

  await user.click(screen.getByRole('button'));
};

describe('Delete Experiment Modal', () => {
  it('submits a valid delete experiment request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: BUTTON_TEXT }));

    expect(mockDeleteExperiment).toHaveBeenCalledWith({ experimentId: experiment.id });
  });
});
