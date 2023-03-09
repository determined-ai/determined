import { render, screen, waitFor } from '@testing-library/react';
import React, { useCallback, useEffect, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';
import { StoreProvider } from 'shared/contexts/stores/UI';
import history from 'shared/routes/history';
import { setAuth, setAuthChecked } from 'stores/auth';
import usersStore from 'stores/users';
import { DetailedUser } from 'types';

import UserManagement, { CREATE_USER, USER_TITLE } from './UserManagement';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: true,
  username: USERNAME,
};

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

const Container: React.FC = () => {
  const [canceler] = useState(new AbortController());

  const loadUsers = useCallback(() => {
    usersStore.ensureUsersFetched(canceler);
    setAuth({ isAuthenticated: true });
    setAuthChecked();
    usersStore.updateCurrentUser(currentUser.id);
  }, [canceler]);

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
    <StoreProvider>
      <DndProvider backend={HTML5Backend}>
        <Container />
      </DndProvider>
    </StoreProvider>,
  );

describe('UserManagement', () => {
  afterEach(() => jest.clearAllTimers());
  it('should render table/button correct values', async () => {
    setup();

    await waitFor(() => jest.setTimeout(300));
    expect(await screen.findByText(CREATE_USER)).toBeInTheDocument();
    expect(await screen.findByText(USER_TITLE)).toBeInTheDocument();

    expect(await screen.findByText(DISPLAY_NAME)).toBeInTheDocument();
    expect(await screen.findByText(USERNAME)).toBeInTheDocument();
    // await waitFor(() => {
    //   expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
    //   expect(screen.getByText(USERNAME)).toBeInTheDocument();
    // });
  });

  // TODO: make this test case work
  // eslint-disable-next-line jest/no-commented-out-tests
  // it('should render modal for create user when click the button', async () => {
  //   setup();
  //   const user = userEvent.setup();
  //   await user.click(await screen.findByLabelText(CREATE_USER_LABEL));
  //   await waitFor(() => {
  //     expect(screen.getByRole('heading', { name: MODAL_HEADER_LABEL_CREATE })).toBeInTheDocument();
  //   });
  // });
});
