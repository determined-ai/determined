import { render, screen } from '@testing-library/react';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import React, { useCallback, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import Admin from '.';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const CURRENT_USER: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: true,
  username: USERNAME,
};

const mocks = vi.hoisted(() => {
  return {
    canAdministrateUsers: false,
    canAssignRoles: vi.fn(),
  };
});

vi.mock('stores/determinedInfo', async (importOriginal) => {
  const observable = await import('utils/observable');
  const store = {
    info: observable.observable({
      rbacEnabled: true,
    }),
  };
  return {
    ...(await importOriginal<typeof import('stores/determinedInfo')>()),
    default: store,
  };
});

vi.mock('hooks/usePermissions', () => {
  const usePermissions = vi.fn(() => {
    return {
      canAdministrateUsers: mocks.canAdministrateUsers,
      canAssignRoles: mocks.canAssignRoles,
    };
  });
  return {
    default: usePermissions,
  };
});

vi.mock('services/api', () => ({
  getGroups: () =>
    Promise.resolve({
      groups: [],
      pagination: {
        endIndex: 10,
        limit: 0,
        offset: 0,
        startIndex: 0,
        total: 10,
      },
    }),
  getUsers: () => {
    const users: Array<DetailedUser> = [CURRENT_USER];
    return Promise.resolve({ pagination: { total: 1 }, users });
  },
  listRoles: () => Promise.resolve([]),
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
          <Admin />;
        </BrowserRouter>
      </HelmetProvider>
    </SettingsProvider>
  );
};

const setup = () =>
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <DndProvider backend={HTML5Backend}>
          <Container />;
        </DndProvider>
      </ThemeProvider>
    </UIProvider>,
  );

describe('Admin page', () => {
  it('should hide users tab without permissions', () => {
    setup();
    expect(screen.getByText('Admin Settings')).toBeInTheDocument();
    expect(screen.queryByText('Users')).not.toBeInTheDocument();
  });

  it('should render users tab with permissions', async () => {
    mocks.canAdministrateUsers = true;
    setup();
    expect(screen.getByText('Admin Settings')).toBeInTheDocument();
    expect(await screen.findByText('Users (1)')).toBeInTheDocument();
  });
});
