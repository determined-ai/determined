import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import { UIProvider } from 'components/kit/Theme';
import { badgeColorFromState, stateToLabel } from 'constants/states';
import { ResourceState, SlotState } from 'types';
import { generateAlphaNumeric } from 'utils/string';

import Badge, { BadgeProps } from './Badge';

const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

const user = userEvent.setup();

const setup = ({ text, badgeColor = undefined }: BadgeProps = { text: CONTENT }) => {
  return render(
    <UIProvider>
      <Badge badgeColor={badgeColor} text={text} />
    </UIProvider>,
  );
};

describe('Badge', () => {
  it('should display content from children', () => {
    const view = setup();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('should display dynamic content from state prop', async () => {
    const TestComponent = () => {
      const [value, setValue] = useState<SlotState>(SlotState.Free);
      return (
        <UIProvider>
          <button role="button" onClick={() => setValue(SlotState.Running)} />
          <Badge badgeColor={badgeColorFromState(value)} text={stateToLabel(value)} />
        </UIProvider>
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
      text: label,
    });
    const statePotential = view.getByText(label);
    expect(statePotential).toHaveClass('dashed');
  });
});
