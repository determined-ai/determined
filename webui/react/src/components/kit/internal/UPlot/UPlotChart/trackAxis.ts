import uPlot, { Plugin } from 'uplot';

import css from 'components/kit/internal/UPlot/UPlotChart/trackAxis.module.scss';

export const trackAxis = (): Plugin => {
  return {
    hooks: {
      ready: (uPlot: uPlot) => {
        uPlot.root.querySelector('.u-over')?.classList.add(css.trackAxis);
      },
    },
  };
};
