import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import { UIProvider } from 'components/kit/Theme';
import { badgeColorFromState, stateToLabel } from 'constants/states';
import { ResourceState, SlotState } from 'types';
import { generateAlphaNumeric } from 'utils/string';

import Badge, { BadgeProps } from './Badge';

const CONTENT = generateAlphaNumeric();

const user = userEvent.setup();

const setup = (props: BadgeProps = { text: CONTENT }) => {
  return render(
    <UIProvider>
      <Badge {...props} />
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

    await user.click(view.getByRole('button'));

    await waitFor(() => {
      expect(view.getByText(stateToLabel(SlotState.Running))).toBeInTheDocument();
    });
  });

  it('should display correct style for potential', () => {
    const label = stateToLabel(ResourceState.Potential);
    const view = setup({
      dashed: true,
      text: label,
    });
    const statePotential = view.getByText(label);
    expect(statePotential).toHaveClass('base dashed');
  });
});
