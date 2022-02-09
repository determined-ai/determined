import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';

import Avatar from './Avatar';

const initName = { initials: 'BB', name: 'Bugs Bunny' };

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

const setup = (name: string) => {
  const handleOnChange = jest.fn();
  const view = render(
    <Avatar hideTooltip={false} name={name} />,
  );
  return { handleOnChange, view };
};

describe('Avatar', () => {
  it('displays initials of name', () => {
    setup(initName.name);

    expect(screen.getByText(initName.initials)).toBeInTheDocument();
  });

  it('displays name on hover', async () => {
    const { view } = setup(initName.name);
    fireEvent.mouseOver(await view.findByText(initName.initials));
    expect(await screen.getByText(initName.name)).toBeInTheDocument();
  });
});
