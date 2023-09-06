import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { StoreProvider } from 'components/kit/contexts/UI';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import UserManagement, { CREATE_USER, USER_TITLE } from './UserManagement';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const CURRENT_USER: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: true,
  username: USERNAME,
};
vi.mock('services/api', () => ({
  getGroups: () => Promise.resolve({ groups: [] }),
  getUserRoles: () => Promise.resolve([]),
  getUsers: () => {
    const users: Array<DetailedUser> = [CURRENT_USER];
    return Promise.resolve({ pagination: { total: 1 }, users });
  },
  getUserSetting: () => Promise.resolve({ settings: [] }),
}));

vi.mock('hooks/useTelemetry', () => ({
  telemetryInstance: {
    track: vi.fn(),
    trackPage: vi.fn(),
    updateTelemetry: vi.fn(),
  },
}));

const Container: React.FC = () => {
  const loadUsers = useCallback(() => {
    userStore.fetchUsers();
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  return (
    <SettingsProvider>
      <HelmetProvider>
        <BrowserRouter>
          <UserManagement />;
        </BrowserRouter>
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
  afterEach(() => {
    vi.clearAllTimers();
  });
  it('should render table/button correct values', async () => {
    setup();

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
  // it('should render modal for create user when click the button', async () => {
  //   setup();
  //   const user = userEvent.setup();
  //   await user.click(await screen.findByLabelText(CREATE_USER_LABEL));
  //   await waitFor(() => {
  //     expect(screen.getByRole('heading', { name: MODAL_HEADER_LABEL_CREATE })).toBeInTheDocument();
  //   });
  // });
});
