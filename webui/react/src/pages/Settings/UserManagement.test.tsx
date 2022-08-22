import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { Router } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'shared/routes/history';
import { DetailedUser } from 'types';

import UserManagement, { CREAT_USER_LABEL, CREATE_USER, USER_TITLE } from './UserManagement';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const user = userEvent.setup();

jest.mock('services/api', () => ({
  getUsers: () => {
    const currentUser: DetailedUser = {
      displayName: DISPLAY_NAME,
      id: 1,
      isActive: true,
      isAdmin: false,
      username: USERNAME,
    };
    const users: Array<DetailedUser> = [ currentUser ];
    return Promise.resolve({ pagination: { total: 1 }, users });
  },
}));

const setup = () => render(
  <StoreProvider>
    <DndProvider backend={HTML5Backend}>
      <HelmetProvider>
        <Router history={history}>
          <UserManagement />
        </Router>
      </HelmetProvider>
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
