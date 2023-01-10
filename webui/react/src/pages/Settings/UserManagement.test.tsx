import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import { MODAL_HEADER_LABEL_CREATE } from 'hooks/useModal/UserSettings/useModalCreateUser';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import { StoreProvider } from 'shared/contexts/stores/UI';
import history from 'shared/routes/history';
import { AuthProvider, useAuth } from 'stores/auth';
import { DeterminedInfoProvider, initInfo, useUpdateDeterminedInfo } from 'stores/determinedInfo';
import { UserRolesProvider } from 'stores/userRoles';
import { useFetchUsers, UsersProvider, useUpdateCurrentUser } from 'stores/users';
import { DetailedUser } from 'types';

import UserManagement, { CREATE_USER, CREATE_USER_LABEL, USER_TITLE } from './UserManagement';

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
  const updateCurrentUser = useUpdateCurrentUser();
  const [canceler] = useState(new AbortController());
  const fetchUsers = useFetchUsers(canceler);
  const { setAuth, setAuthCheck } = useAuth();
  const updateInfo = useUpdateDeterminedInfo();

  const loadUsers = useCallback(async () => {
    await fetchUsers();
    setAuth({ isAuthenticated: true });
    setAuthCheck();
    updateCurrentUser(currentUser.id);
    updateInfo({ ...initInfo, featureSwitches: [], rbacEnabled: false });
  }, [fetchUsers, setAuthCheck, updateCurrentUser, setAuth, updateInfo]);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  return (
    <SettingsProvider>
      <HelmetProvider>
        <HistoryRouter history={history}>
          <UserManagement />;
        </HistoryRouter>
      </HelmetProvider>
    </SettingsProvider>
  );
};

const setup = () =>
  render(
    <DeterminedInfoProvider>
      <StoreProvider>
        <UsersProvider>
          <AuthProvider>
            <UserRolesProvider>
              <DndProvider backend={HTML5Backend}>
                <Container />
              </DndProvider>
            </UserRolesProvider>
          </AuthProvider>
        </UsersProvider>
      </StoreProvider>
    </DeterminedInfoProvider>,
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
    await user.click(await screen.findByLabelText(CREATE_USER_LABEL));
    expect(screen.getByRole('heading', { name: MODAL_HEADER_LABEL_CREATE })).toBeInTheDocument();
  });
});
