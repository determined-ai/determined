import uPlot, { Plugin } from 'uplot';

import { glasbeyColor } from 'utils/color';

import css from './tooltipsPlugin.module.scss';

export type ChartTooltipData = (string | null)[];

interface Props {
  getTooltipTextForXIndex?: (xIndex: number) => ChartTooltipData;
}

export const tooltipsPlugin = ({ getTooltipTextForXIndex }: Props = {}): Plugin => {
  let barEl: HTMLDivElement|null = null;
  let displayedIdx: number|null = null;
  let tooltipEl: HTMLDivElement|null = null;

  const _buildTooltipHtml = (uPlot: uPlot, idx: number): string => {
    let html = '';

    const xSerie = uPlot.series[0];
    const xValue = (typeof xSerie.value === 'function' ?
      xSerie.value(uPlot, uPlot.data[0][idx], 0, idx) : uPlot.data[0][idx]);
    html += `<div class="${css.valueX}">`
      + `${xSerie.label}: ${xValue}`
      + '</div>';

    let tooltipForXIndex: ChartTooltipData = [];
    if (typeof getTooltipTextForXIndex === 'function') {
      tooltipForXIndex = getTooltipTextForXIndex(idx);
    }

    uPlot.series.forEach((serie, i) => {
      if (serie.scale === 'x' || !serie.show) return;

      const tooltipText = tooltipForXIndex[i - 1] || null;
      const valueRaw = uPlot.data[i][idx];

      const cssClass = valueRaw ? css.valueY : css.valueYEmpty;
      html += `<div class="${cssClass}">`
        + `<span class="${css.color}" style="background-color: ${glasbeyColor(i - 1)}"></span>`
        + `${serie.label}: ${valueRaw || 'N/A'}`
        + (tooltipText ? '<br />' + tooltipText : '')
        + '</div>';
    });

    return html;
  };

  const _getTooltipLeftPx = (uPlot: uPlot, idx: number): number => {
    const idxLeft = uPlot.valToPos(uPlot.data[0][idx], 'x');
    if (!tooltipEl) return idxLeft;

    const chartWidth = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().width;
    const tooltipWidth = tooltipEl.getBoundingClientRect().width;

    // right
    if (chartWidth && idxLeft + tooltipWidth >= chartWidth) {
      return (idxLeft - tooltipWidth);
    }

    // left
    return idxLeft;
  };

  const _updateTooltipVerticalPosition = (uPlot: uPlot, cursorTop: number) => {
    if (!tooltipEl) return;

    const chartHeight = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().height;

    const vPos = (chartHeight && cursorTop > (chartHeight/2)) ? 'top' : 'bottom';

    tooltipEl.style.bottom = vPos === 'bottom' ? '0px' : 'auto';
    tooltipEl.style.top = vPos === 'top' ? '0px' : 'auto';
  };

  const showIdx = (uPlot: uPlot, idx: number) => {
    if (!tooltipEl || !barEl) return;
    displayedIdx = idx;

    const idxLeft = uPlot.valToPos(uPlot.data[0][idx], 'x');

    barEl.style.display = 'block';
    barEl.style.left = idxLeft + 'px';

    tooltipEl.innerHTML = _buildTooltipHtml(uPlot, idx);
    tooltipEl.style.display = 'block';
    tooltipEl.style.left = _getTooltipLeftPx(uPlot, idx) + 'px';
  };

  const hide = () => {
    if (!tooltipEl || !barEl) return;
    displayedIdx = null;

    barEl.style.display = 'none';
    tooltipEl.style.display = 'none';
  };

  return {
    hooks: {
      ready: (uPlot: uPlot) => {
        tooltipEl = document.createElement('div');
        tooltipEl.className = css.tooltip;
        uPlot.root.querySelector('.u-over')?.appendChild(tooltipEl);

        barEl = document.createElement('div');
        barEl.className = css.bar;
        uPlot.root.querySelector('.u-over')?.appendChild(barEl);

      },
      setCursor: (uPlot: uPlot) => {
        const { left, idx, top } = uPlot.cursor;

        if (
          (idx == null && displayedIdx)
          || !left || left < 0
          || !top || top < 0
        ) {
          hide();
          return;
        }

        if (idx != null && idx !== displayedIdx) {
          showIdx(uPlot, idx);
        }

        _updateTooltipVerticalPosition(uPlot, top);
      },
    },
  };
};
