import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import { stateToLabel } from 'constants/states';
import { generateAlphaNumeric } from 'shared/utils/string';
import { ResourceState, SlotState } from 'types';

import Badge, { BadgeType } from './Badge';

const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

const setup = () => {
  const view = render(<Badge tooltip={CONTENT_TOOLTIP} type={BadgeType.Header}>{CONTENT}</Badge>);
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
        <>
          <button role="button" onClick={() => setValue(SlotState.Running)} />
          <Badge state={value} type={BadgeType.State} />
        </>
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
    const BadgeComponent = () => {
      return <Badge state={ResourceState.Potential} type={BadgeType.State} />;
    };
    const view = render(<BadgeComponent />);
    const statePotential = view.getByText(stateToLabel(ResourceState.Potential));
    expect(statePotential).toHaveClass('state neutral dashed');
  });
});
