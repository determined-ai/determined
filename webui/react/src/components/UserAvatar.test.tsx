import { waitFor } from '@testing-library/dom';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ThemeProvider, UIProvider } from 'components/kit/Theme';
import { User } from 'types';
import { isDarkMode, theme } from 'utils/tests/getTheme';

import UserAvatar, { Props } from './UserAvatar';

const testUsers: User[] = [
  {
    displayName: 'Bugs Bunny',
    id: 44,
    username: 'elmerFudd01',
  },
];

vi.mock('components/kit/Tooltip');

const Component = ({ user }: Partial<Props> = {}) => {
  return <UserAvatar hideTooltip={false} user={user} />;
};

const setup = (testUser: User) => {
  const user = userEvent.setup();
  const view = render(
    <ThemeProvider>
      <UIProvider darkMode={isDarkMode} theme={theme}>
        <Component user={testUser} />
      </UIProvider>
    </ThemeProvider>,
  );

  return { user, view };
};

describe('UserAvatar', () => {
  it('should display initials of name', async () => {
    const testUser = testUsers[0];
    await waitFor(() => setup(testUser));
    expect(await screen.findByText('BB')).toBeInTheDocument();
  });

  it('should display name on hover', async () => {
    const testUser = testUsers[0];
    const { user } = await waitFor(() => setup(testUser));
    await act(async () => await user.hover(await screen.findByText('BB')));
    expect(await screen.getByText(testUser.displayName || '')).toBeInTheDocument();
  });
});
