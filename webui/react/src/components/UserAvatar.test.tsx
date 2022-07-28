import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProps } from 'antd/es/tooltip';
import React from 'react';

import StoreProvider from 'contexts/Store';

import UserAvatar, { Props } from './UserAvatar';

const testUser = {
  displayName: 'Bugs Bunny',
  id: 44,
  initials: 'BB',
  userId: 44,
  username: 'elmerFudd01',
};

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** We need to mock Tooltip in order to override getPopupContainer to null. getPopupContainer
   * sets the DOM container and if this prop is set, the popup div may not be available in the body
   */
  const Tooltip = (props: TooltipProps) => {
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
    return Promise.resolve({ users: [ testUser ] });
  },
}));

const user = userEvent.setup();

const setup = ({ hideTooltip = false, userId, ...props }: Partial<Props> = {}) => {
  render(
    <StoreProvider>
      <UserAvatar hideTooltip={hideTooltip} userId={userId} {...props} />
    </StoreProvider>,
  );
};

describe('UserAvatar', () => {
  it('should display initials of name', async () => {
    setup(testUser);

    expect(await screen.findByText(testUser.initials)).toBeInTheDocument();
  });

  it('should display name on hover', async () => {
    setup(testUser);

    await user.hover(await screen.findByText(testUser.initials));

    expect(await screen.findByText(testUser.displayName)).toBeInTheDocument();
  });
});
