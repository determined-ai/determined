import { fireEvent, render, screen } from '@testing-library/react';
import { TooltipProps } from 'antd/es/tooltip';
import React from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { Metric } from 'types';

import MetricBadgeTag from './MetricBadgeTag';

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** mocking Tooltip based on Avatar test */
  const Tooltip = (props: TooltipProps) => {
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

const setup = (metric: Metric) => {
  const handleOnChange = jest.fn();
  const view = render(
    <UIProvider>
      <MetricBadgeTag metric={metric} />,
    </UIProvider>,
  );
  return { handleOnChange, view };
};

describe('MetricBadgeTag', () => {
  const sampleMetric: Metric = {
    name: 'accuracy',
    type: 'validation',
  };

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
