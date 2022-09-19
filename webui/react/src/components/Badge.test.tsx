import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import { stateToLabel } from 'constants/states';
import StoreProvider from 'contexts/Store';
import { generateAlphaNumeric } from 'shared/utils/string';
import { ResourceState, SlotState } from 'types';

import Badge, { BadgeProps, BadgeType } from './Badge';

const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

const user = userEvent.setup();

const setup = ({
  children = CONTENT,
  tooltip = CONTENT_TOOLTIP,
  type = BadgeType.Header,
  ...props
}: BadgeProps = {}) => {
  return render(
    <StoreProvider>
      <Badge tooltip={tooltip} type={type} {...props}>{children}</Badge>
    </StoreProvider>,
  );
};

describe('Badge', () => {
  it('should display content from children', () => {
    const view = setup();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('should display dynamic content from state prop', async () => {
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

    await user.click(view.getByRole('button'));

    await waitFor(() => {
      expect(view.getByText(stateToLabel(SlotState.Running))).toBeInTheDocument();
    });
  });

  it('should apply className by type', () => {
    const view = setup();
    expect(view.getByText(CONTENT)).toHaveClass('header');
  });

  it('should display tooltip on hover', async () => {
    const view = setup();
    await user.hover(view.getByText(CONTENT));
    await waitFor(() => {
      expect(view.getByRole('tooltip').textContent).toEqual(CONTENT_TOOLTIP);
    });
  });

  it('should display correct style for potential', () => {
    const label = stateToLabel(ResourceState.Potential);
    const view = setup({
      children: label,
      state: ResourceState.Potential,
      type: BadgeType.State,
    });
    const statePotential = view.getByText(label);
    expect(statePotential).toHaveClass('state neutral dashed');
  });
});
