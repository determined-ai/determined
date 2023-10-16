import { fireEvent, render, screen } from '@testing-library/react';

import { UIProvider, ThemeProvider } from 'components/kit/Theme';
import { Metric } from 'types';
import { theme, isDarkMode } from 'utils/tests/getTheme';
import MetricBadgeTag from './MetricBadgeTag';

vi.mock('components/kit/Tooltip');

const setup = (metric: Metric) => {
  const handleOnChange = vi.fn();
  const view = render(
    <ThemeProvider>
      <UIProvider theme={theme} darkMode={isDarkMode}>
        <MetricBadgeTag metric={metric} />
      </UIProvider>
    </ThemeProvider>
  );
  return { handleOnChange, view };
};

describe('MetricBadgeTag', () => {
  const sampleMetric: Metric = { group: 'validation', name: 'accuracy' };

  it('displays metric name and first letter of type', () => {
    setup(sampleMetric);
    expect(screen.getByText('accuracy')).toBeInTheDocument();
    expect(screen.getByText('V')).toBeInTheDocument();
  });

  it('displays name on hover', async () => {
    const { view } = setup(sampleMetric);
    fireEvent.mouseOver(await view.findByText('V'));
    expect(await screen.getByText('validation')).toBeInTheDocument();
  });
});
