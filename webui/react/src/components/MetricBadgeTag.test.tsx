import { fireEvent, render, screen } from '@testing-library/react';

import MetricBadgeTag from 'components/MetricBadgeTag';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';
import { Metric } from 'types';

vi.mock('components/kit/Tooltip');

const setup = (metric: Metric) => {
  const handleOnChange = vi.fn();
  const view = render(
    <UIProvider>
      <MetricBadgeTag metric={metric} />,
    </UIProvider>,
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
