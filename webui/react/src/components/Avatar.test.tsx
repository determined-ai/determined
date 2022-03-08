import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import StoreProvider from 'contexts/Store';

import Avatar from './Avatar';

const testUser = { displayName: 'Bugs Bunny', initials: 'BB', username: 'elmerFudd01' };
const mockUsers = [ testUser ];

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** We need to mock Tooltip in order to override getPopupContainer to null. getPopupContainer
   * sets the DOM container and if this prop is set, the popup div may not be available in the body
   */
  const Tooltip = (props: unknown) => {
    return (
      <antd.Tooltip
        {...props}
        getPopupContainer={(trigger: HTMLElement) => trigger}
        mouseEnterDelay={0}
      />
    );
  };

  return {
    __esModule: true,
    ...antd,
    Tooltip,
  };
});

jest.mock('services/api', () => ({
  getUsers: () => {
    return Promise.resolve(mockUsers);
  },
}));

const TestApp: React.FC = () => {
  return (
    <div>
      <Avatar hideTooltip={false} username={testUser.username} />
    </div>
  );
};

const setup = () => {
  render(
    <StoreProvider>
      <TestApp />
    </StoreProvider>,
  );
};

describe('Avatar', () => {
  it('displays initials of name', async () => {
    setup();

    await waitFor(() => {
      expect(screen.getByText(testUser.initials)).toBeInTheDocument();
    });
  });

  it('displays name on hover', async () => {
    setup();

    await waitFor(() => {
      userEvent.hover(screen.getByText(testUser.initials));
      expect(screen.getByText(testUser.displayName)).toBeInTheDocument();
    });
  });
});
