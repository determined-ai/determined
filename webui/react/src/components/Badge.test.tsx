import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useState } from 'react';

import { ThemeProvider, UIProvider } from 'components/kit/Theme';
import { stateToLabel } from 'constants/states';
import { ResourceState, SlotState } from 'types';
import { generateAlphaNumeric } from 'utils/string';
import { isDarkMode, theme } from 'utils/tests/getTheme';

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
    <ThemeProvider>
      <UIProvider darkMode={isDarkMode} theme={theme}>
        <Badge tooltip={tooltip} type={type} {...props}>
          {children}
        </Badge>
      </UIProvider>
    </ThemeProvider>,
  );
};

describe('Badge', () => {
  it('should display content from children', () => {
    const view = setup();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('should display dynamic content from state prop', () => {
    const TestComponent = () => {
      const [value, setValue] = useState<SlotState>(SlotState.Free);
      return (
        <ThemeProvider>
          <UIProvider darkMode={isDarkMode} theme={theme}>
            <button role="button" onClick={() => setValue(SlotState.Running)} />
            <Badge state={value} type={BadgeType.State} />
          </UIProvider>
        </ThemeProvider>
      );
    };

    const view = render(<TestComponent />);
    const slotFree = view.getByText(stateToLabel(SlotState.Free));

    expect(slotFree).toHaveClass('state neutral');

    user.click(view.getByRole('button'));

    waitFor(() => {
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
