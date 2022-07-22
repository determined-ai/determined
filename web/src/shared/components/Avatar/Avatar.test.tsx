import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProps } from 'antd/es/tooltip';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { DarkLight } from 'shared/themes';

import Avatar, { Props } from './Avatar';

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

const user = userEvent.setup();

const setup = ({
  darkLight = DarkLight.Light,
  displayName = 'Anonymous',
  hideTooltip = false,
  ...props
}: Partial<Props> = {}) => {
  render(
    <StoreProvider>
      <Avatar
        darkLight={darkLight}
        displayName={displayName}
        hideTooltip={hideTooltip}
        {...props}
      />
    </StoreProvider>,
  );
};

describe('Avatar', () => {
  const testUser = {
    displayName: 'Bugs Bunny',
    initials: 'BB',
    username: 'elmerFudd01',
  };

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
