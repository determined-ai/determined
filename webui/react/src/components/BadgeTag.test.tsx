import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ThemeProvider, UIProvider } from 'components/kit/Theme';
import { generateAlphaNumeric } from 'utils/string';
import { theme, isDarkMode } from 'utils/tests/getTheme';
import BadgeTag, { Props } from './BadgeTag';

const LABEL = generateAlphaNumeric();
const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

vi.mock('components/kit/Tooltip');

const setup = ({ children = CONTENT, tooltip = CONTENT_TOOLTIP, ...props }: Props = {}) => {

  const view = render(
    <ThemeProvider>
      <UIProvider theme={theme} darkMode={isDarkMode}>
        <BadgeTag tooltip={tooltip} {...props}>
          {children}
        </BadgeTag>
      </UIProvider>
    </ThemeProvider>
  );
  return { view };
};

describe('BadgeTag', () => {
  it('displays label and content', () => {
    const { view } = setup({ label: LABEL });
    expect(view.getByText(LABEL)).toBeInTheDocument();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('displays prelabel', () => {
    const { view } = setup({ preLabel: LABEL });
    expect(view.getByText(LABEL)).toBeInTheDocument();
  });

  it('label displays tooltip on hover', async () => {
    const { view } = setup({ label: LABEL });
    userEvent.hover(view.getByText(LABEL));
    expect((await view.findByRole('tooltip')).textContent).toEqual(LABEL);
  });

  it('content displays tooltip on hover', async () => {
    const { view } = setup({ label: LABEL });
    userEvent.hover(view.getByText(CONTENT));
    expect((await view.findByRole('tooltip')).textContent).toEqual(CONTENT_TOOLTIP);
  });
});
