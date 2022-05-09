import { render } from '@testing-library/react';
import React from 'react';
import uPlot from 'uplot';

import UPlotChart from './UPlotChart';

const DEFAULT_SIZE = { height: 1024, width: 1280, x: 0, y: 0 };

jest.mock('hooks/useResize', () => ({ __esModule: true, default: () => DEFAULT_SIZE }));

const setup = (options?: Partial<uPlot.Options>, data?: uPlot.AlignedData) => {
  const view = render(<UPlotChart data={data} options={options} />);
  return view;
};

describe('UPlotChart', () => {
  const data: uPlot.AlignedData = [ [ 0, 100, 200 ], [ 1, 2, 3 ] ];
  const options: Partial<uPlot.Options> = { title: 'Chart Title' };

  it('should render chart with title', () => {
    const view = setup(options, data);
    expect(view.getByText(options.title ?? '')).toBeInTheDocument();
  });
});
