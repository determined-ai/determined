import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect, useMemo } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import history from 'shared/routes/history';
import { DetailedUser } from 'types';

import UserManagement, { CREAT_USER_LABEL, CREATE_USER, USER_TITLE } from './UserManagement';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const user = userEvent.setup();

jest.mock('services/api', () => ({
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
  useStore: () => ({ auth: { checked: true, user: { id: 1 } as DetailedUser }, info: { featureSwitches: [], rbacEnabled: false } }),
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
  const storeDispatch = useStoreDispatch();

  const currentUser: DetailedUser = useMemo(
    () => ({
      displayName: DISPLAY_NAME,
      id: 1,
      isActive: true,
      isAdmin: true,
      username: USERNAME,
    }),
    [],
  );

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: [currentUser] });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: currentUser });
  }, [storeDispatch, currentUser]);

  useEffect(() => loadUsers(), [loadUsers]);

  return <UserManagement />;
};

const setup = () =>
  render(
    <StoreProvider>
      <DndProvider backend={HTML5Backend}>
        <SettingsProvider>
          <HelmetProvider>
            <HistoryRouter history={history}>
              <Container />
            </HistoryRouter>
          </HelmetProvider>
        </SettingsProvider>
      </DndProvider>
    </StoreProvider>,
  );

describe('UserManagement', () => {
  it('should render table/button correct values', async () => {
    await waitFor(() => setup());

    expect(screen.getByText(CREATE_USER)).toBeInTheDocument();
    expect(screen.getByText(USER_TITLE)).toBeInTheDocument();
    expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
    expect(screen.getByText(USERNAME)).toBeInTheDocument();
  });

  it('should render modal for create user when click the button', async () => {
    await waitFor(() => setup());
    await user.click(screen.getByLabelText(CREAT_USER_LABEL));

    expect(screen.getAllByText('Create User')).toHaveLength(2);
  });
});
