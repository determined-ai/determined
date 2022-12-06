import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import history from 'shared/routes/history';
import { AuthProvider, useAuth } from 'stores/auth';
import { useFetchUsers, UsersProvider } from 'stores/users';
import { DetailedUser } from 'types';

import UserManagement, { CREAT_USER_LABEL, CREATE_USER, USER_TITLE } from './UserManagement';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const user = userEvent.setup();

jest.mock('services/api', () => ({
  ...jest.requireActual('services/api'),
  getGroups: () => Promise.resolve({ groups: [] }),
  getUserRoles: () => Promise.resolve([]),
  getUsers: () => {
    const currentUser: DetailedUser = {
      displayName: DISPLAY_NAME,
      id: 1,
      isActive: true,
      isAdmin: true,
      username: USERNAME,
    };
    const users: Array<DetailedUser> = [currentUser];
    return Promise.resolve({ pagination: { total: 1 }, users });
  },
  getUserSetting: () => Promise.resolve({ settings: [] }),
}));

jest.mock('contexts/Store', () => ({
  __esModule: true,
  ...jest.requireActual('contexts/Store'),
  useStore: () => ({
    auth: { checked: true, user: { id: 1 } as DetailedUser },
    info: { featureSwitches: [], rbacEnabled: false },
  }),
}));

jest.mock('hooks/useTelemetry', () => ({
  ...jest.requireActual('hooks/useTelemetry'),
  telemetryInstance: {
    track: jest.fn(),
    trackPage: jest.fn(),
    updateTelemetry: jest.fn(),
  },
}));

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: true,
  username: USERNAME,
};

const Container: React.FC = () => {
  const { updateCurrentUser } = useAuth();
  const [canceler] = useState(new AbortController());
  const fetchUsers = useFetchUsers(canceler);

  const loadUsers = useCallback(async () => {
    await fetchUsers();

    updateCurrentUser(currentUser, [currentUser]);
  }, [fetchUsers, updateCurrentUser]);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  return <UserManagement />;
};

const setup = () =>
  render(
    <StoreProvider>
      <UsersProvider>
        <AuthProvider>
          <DndProvider backend={HTML5Backend}>
            <SettingsProvider>
              <HelmetProvider>
                <HistoryRouter history={history}>
                  <Container />
                </HistoryRouter>
              </HelmetProvider>
            </SettingsProvider>
          </DndProvider>
        </AuthProvider>
      </UsersProvider>
    </StoreProvider>,
  );

describe('UserManagement', () => {
  afterEach(() => jest.clearAllTimers());
  it('should render table/button correct values', async () => {
    setup();

    await waitFor(() => jest.setTimeout(300));
    expect(await screen.findByText(CREATE_USER)).toBeInTheDocument();
    expect(await screen.findByText(USER_TITLE)).toBeInTheDocument();
    waitFor(() => {
      expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
      expect(screen.getByText(USERNAME)).toBeInTheDocument();
    });
  });

  it('should render modal for create user when click the button', async () => {
    setup();
    await user.click(await screen.findByLabelText(CREAT_USER_LABEL));
    expect(screen.getAllByText('New User')).toHaveLength(1);
  });
});
