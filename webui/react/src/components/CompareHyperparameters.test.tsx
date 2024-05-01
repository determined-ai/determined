//

import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { NO_DATA_MESSAGE } from './CompareHyperparameters';
import { CompareHyperparametersWithMocks } from './CompareHyperparameters.test.mock';
import { ThemeProvider } from './ThemeProvider';
const setup = (empty?: boolean) => {
  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <CompareHyperparametersWithMocks empty={empty} />
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );
};

const PARALLEL_COORDINATES = 'Parallel Coordinates';
const SCATTER_PLOTS = 'Scatter Plots';
const HEAT_MAP = 'Heat Map';

describe('CompareHyperparameters component', () => {
  it(`renders ${PARALLEL_COORDINATES}`, () => {
    setup();
    expect(screen.queryByText(PARALLEL_COORDINATES)).toBeInTheDocument();
  });
  it(`renders ${SCATTER_PLOTS}`, () => {
    setup();
    expect(screen.queryByText(SCATTER_PLOTS)).toBeInTheDocument();
  });
  it(`renders ${HEAT_MAP}`, () => {
    setup();
    expect(screen.queryByText(HEAT_MAP)).toBeInTheDocument();
  });
  it('renders no data state', () => {
    setup(true);
    expect(screen.queryByText(PARALLEL_COORDINATES)).not.toBeInTheDocument();
    expect(screen.queryByText(SCATTER_PLOTS)).not.toBeInTheDocument();
    expect(screen.queryByText(HEAT_MAP)).not.toBeInTheDocument();
    expect(screen.queryByText(NO_DATA_MESSAGE)).toBeInTheDocument();
  });
});
