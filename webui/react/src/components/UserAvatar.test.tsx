import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProps } from 'antd/es/tooltip';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';

import UserAvatar, { Props } from './UserAvatar';

const testUsers = [
  {
    displayName: 'Bugs Bunny',
    id: 44,
    initials: 'BB',
    isActive: true,
    isAdmin: true,
    userId: 44,
    username: 'elmerFudd01',
  },
];

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

const Component = ({ hideTooltip = false, userId, ...props }: Partial<Props> = {}) => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: testUsers });
  }, [ storeDispatch ]);

  return <UserAvatar hideTooltip={hideTooltip} userId={userId} {...props} />;
};

const setup = ({ hideTooltip = false, userId, ...props }: Partial<Props> = {}) => {
  const user = userEvent.setup();

  const view = render(
    <StoreProvider>
      <Component hideTooltip={hideTooltip} userId={userId} {...props} />
    </StoreProvider>,
  );

  return { user, view };
};

describe('UserAvatar', () => {
  it('should display initials of name', async () => {
    const testUser = testUsers[0];
    setup(testUser);
    expect(await screen.findByText(testUser.initials)).toBeInTheDocument();
  });

  it('should display name on hover', async () => {
    const testUser = testUsers[0];
    const { user } = setup(testUser);
    await user.hover(await screen.findByText(testUser.initials));
    expect(await screen.findByText(testUser.displayName)).toBeInTheDocument();
  });
});
