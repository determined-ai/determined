import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';

import MetricBadgeTag from './MetricBadgeTag';

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** mocking Tooltip based on Avatar test */
  const Tooltip = (props) => {
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

const setup = (metric) => {
  const handleOnChange = jest.fn();
  const view = render(
    <MetricBadgeTag metric={metric} />,
  );
  return { handleOnChange, view };
};

describe('MetricBadgeTag', () => {
  const sampleMetric = {
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
