import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useInitApi } from 'hew/Toast';
import { useState } from 'react';

import { login } from 'services/api';

import DeterminedAuth, {
  PASSWORD_ID,
  SUBMIT_ID,
  USERNAME_ID,
  WEAK_PASSWORD_SUBJECT,
} from './DeterminedAuth';
import { ThemeProvider } from './ThemeProvider';

const USERNAME = 'username';
const WEAK_PASSWORD = 'password';
const STRONG_PASSWORD = 'dO2Ccf5ceozXJFuMhjEx32UgtR4X4yZTm1Vv';

vi.mock('services/api', () => ({
  login: vi.fn().mockResolvedValue({ token: '', user: {} }),
}));

const Container = (): JSX.Element => {
  const [canceler] = useState(new AbortController());
  useInitApi();
  return <DeterminedAuth canceler={canceler} />;
};

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <Container />
      </ThemeProvider>
    </UIProvider>,
  );
};

const user = userEvent.setup();
describe('DeterminedAuth', () => {
  it('logs in with strong password', async () => {
    setup();

    await user.type(screen.getByTestId(USERNAME_ID), USERNAME);
    await user.type(screen.getByTestId(PASSWORD_ID), STRONG_PASSWORD);
    user.click(screen.getByTestId(SUBMIT_ID));

    await waitFor(() => {
      // onFinish begins:
      expect(screen.getByRole('button')).toBeDisabled();
    });
    await waitFor(() => {
      // onFinish ends:
      expect(screen.getByRole('button')).not.toBeDisabled();

      // not.toBeInTheDocument is only a valid assertion after onFinish begins and ends:
      expect(screen.queryByText(WEAK_PASSWORD_SUBJECT)).not.toBeInTheDocument();
      expect(vi.mocked(login)).toHaveBeenLastCalledWith(
        { password: STRONG_PASSWORD, username: USERNAME },
        {
          signal: expect.any(AbortSignal),
        },
      );
    });
  });

  it('logs in with warning when using a weak password', async () => {
    setup();
    await user.clear(screen.getByTestId(USERNAME_ID));
    await user.type(screen.getByTestId(USERNAME_ID), USERNAME);
    await user.type(screen.getByTestId(PASSWORD_ID), WEAK_PASSWORD);
    user.click(screen.getByTestId(SUBMIT_ID));

    await waitFor(() => {
      expect(screen.getByText(WEAK_PASSWORD_SUBJECT)).toBeInTheDocument();
      expect(vi.mocked(login)).toHaveBeenLastCalledWith(
        { password: WEAK_PASSWORD, username: USERNAME },
        {
          signal: expect.any(AbortSignal),
        },
      );
    });
  });
});
