import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { PropsWithChildren, useState } from 'react';

import { stateToLabel } from 'constants/states';
import StoreProvider from 'contexts/Store';
import { generateAlphaNumeric } from 'shared/utils/string';
import { ResourceState, SlotState } from 'types';

import Badge, { BadgeProps, BadgeType } from './Badge';

const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

const setup = ({
  children = CONTENT,
  tooltip = CONTENT_TOOLTIP,
  type = BadgeType.Header,
  ...props
}: PropsWithChildren<BadgeProps> = {}) => {
  const view = render(
    <StoreProvider>
      <Badge tooltip={tooltip} type={type} {...props}>{children}</Badge>
    </StoreProvider>,
  );
  return { view };
};

describe('Badge', () => {
  it('displays content from children', () => {
    const { view } = setup();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('displays dynamic content from state prop', async () => {
    const TestComponent = () => {
      const [ value, setValue ] = useState(SlotState.Free);
      return (
        <StoreProvider>
          <button role="button" onClick={() => setValue(SlotState.Running)} />
          <Badge state={value} type={BadgeType.State} />
        </StoreProvider>
      );
    };
    const view = render(<TestComponent />);
    const slotFree = await view.getByText(stateToLabel(SlotState.Free));

    expect(slotFree).toHaveClass('state neutral');

    userEvent.click(view.getByRole('button'));

    await waitFor(() => {
      expect(view.getByText(stateToLabel(SlotState.Running))).toBeInTheDocument();
    });
  });

  it('applies className by type', () => {
    const { view } = setup();
    expect(view.getByText(CONTENT)).toHaveClass('header');
  });

  it('displays tooltip on hover', async () => {
    const { view } = setup();
    userEvent.hover(view.getByText(CONTENT));
    await waitFor(() => {
      expect(view.getByRole('tooltip').textContent).toEqual(CONTENT_TOOLTIP);
    });
  });

  it('displays correct style for potential', () => {
    const label = stateToLabel(ResourceState.Potential);
    const { view } = setup({
      children: label,
      state: ResourceState.Potential,
      type: BadgeType.State,
    });
    const statePotential = view.getByText(label);
    expect(statePotential).toHaveClass('state neutral dashed');
  });
});
