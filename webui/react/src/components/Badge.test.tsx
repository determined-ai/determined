import { fireEvent, render, screen } from '@testing-library/react';
import React, { useState } from 'react';

import { stateToLabel } from 'constants/states';
import { getStateColorCssVar } from 'themes';
import { SlotState } from 'types';

import Badge, { BadgeType } from './Badge';

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  // TODO: move Tooltip mock to shared file
  const Tooltip = (props) => {
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

describe('Badge', () => {
  it('displays content from children', () => {
    render(<Badge>Badge content</Badge>);
    expect(screen.getByText('Badge content')).toBeInTheDocument();
  });

  it('displays dynamic content from state prop', async () => {
    const TestComponent = () => {
      const [ value, setValue ] = useState(SlotState.Free);
      return (
        <>
          <button role="button" onClick={() => setValue(SlotState.Running)} />
          <Badge state={value} type={BadgeType.State} />
        </>
      );
    };
    render(<TestComponent />);
    const slotFree = screen.getByText(stateToLabel(SlotState.Free));
    expect(slotFree).toHaveStyle({
      backgroundColor: getStateColorCssVar(SlotState.Free),
      color: '#234b65',
    });
    fireEvent.click(await screen.getByRole('button'));
    const slotRunning = screen.getByText(stateToLabel(SlotState.Running));
    expect(slotRunning).toHaveStyle({ backgroundColor: getStateColorCssVar(SlotState.Running) });
  });

  it('applies className by type', () => {
    render(<Badge type={BadgeType.Header}>Badge content</Badge>);
    const badge = screen.getByText('Badge content');
    expect(badge).toHaveClass('header');
  });

  it('displays tooltip on hover', async () => {
    render(<Badge tooltip="Tooltip text" type={BadgeType.Header}>Badge content</Badge>);
    fireEvent.mouseOver(await screen.findByText('Badge content'));
    expect(screen.getByText('Tooltip text')).toBeInTheDocument();
  });
});
