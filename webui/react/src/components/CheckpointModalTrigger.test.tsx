import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';
import React, { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import { ThemeProvider } from 'components/ThemeProvider';
import authStore from 'stores/auth';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

const TEST_MODAL_TITLE = 'Checkpoint Modal Test';
const REGISTER_CHECKPOINT_TEXT = 'Register Checkpoint';

vi.mock('services/api', () => ({
  getModels: () => {
    return Promise.resolve({ models: [] });
  },
}));

const user = userEvent.setup();

const ModalTrigger: React.FC = () => {
  const { experiment, checkpoint } = generateTestExperimentData();

  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
  }, []);

  return (
    <CheckpointModalTrigger
      checkpoint={checkpoint}
      experiment={experiment}
      title={TEST_MODAL_TITLE}
    />
  );
};

const setup = async () => {
  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <ConfirmationProvider>
            <ModalTrigger />
          </ConfirmationProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );

  await user.click(screen.getByRole('button'));

  return user;
};

describe('CheckpointModalTrigger', () => {
  it('open modal', async () => {
    await setup();

    expect(await screen.findByText(TEST_MODAL_TITLE)).toBeInTheDocument();
  });

  it('close modal', async () => {
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
      // One for the title and one for the confirm button label.
      const elements = screen.queryAllByText(REGISTER_CHECKPOINT_TEXT);
      expect(elements).toHaveLength(2);
    });
  });
});
