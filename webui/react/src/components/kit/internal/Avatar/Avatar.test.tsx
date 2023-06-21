import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import { DarkLight } from 'components/kit/internal/types';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';

import Avatar, { Props } from './Avatar';

vi.mock('components/kit/Tooltip');
const user = userEvent.setup();

const setup = ({
  darkLight = DarkLight.Light,
  displayName = 'Anonymous',
  hideTooltip = false,
  ...props
}: Partial<Props> = {}) => {
  render(
    <UIProvider>
      <Avatar
        darkLight={darkLight}
        displayName={displayName}
        hideTooltip={hideTooltip}
        {...props}
      />
    </UIProvider>,
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
