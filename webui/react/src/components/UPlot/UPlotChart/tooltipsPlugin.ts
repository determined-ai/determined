import uPlot, { Plugin } from 'uplot';

import css from 'components/UPlot/UPlotChart/tooltipsPlugin.module.scss';
import { glasbeyColor } from 'utils/color';
import { humanReadableNumber } from 'utils/number';

export type ChartTooltip = string | null;

interface TooltipResult {
  html?: string;
  val: number | null | undefined;
  yDist?: number;
}

interface Props {
  closeOnMouseExit?: boolean;
  getXTooltipHeader?: (xIndex: number) => ChartTooltip;
  getXTooltipYLabels?: (xIndex: number) => ChartTooltip[];
  isShownEmptyVal: boolean;
  seriesColors: string[];
}

export const tooltipsPlugin = (
  {
    closeOnMouseExit,
    getXTooltipHeader,
    getXTooltipYLabels,
    isShownEmptyVal,
    seriesColors,
  }: Props = {
    isShownEmptyVal: true,
    seriesColors: [],
  },
): Plugin => {
  let barEl: HTMLDivElement | null = null;
  let displayedIdx: number | null = null;
  let tooltipEl: HTMLDivElement | null = null;

  const _buildTooltipHtml = (uPlot: uPlot, idx: number): string => {
    let html = '';

    const header: ChartTooltip =
      typeof getXTooltipHeader === 'function' ? getXTooltipHeader(idx) : '';

    const yLabels: ChartTooltip[] =
      typeof getXTooltipYLabels === 'function' ? getXTooltipYLabels(idx) : [];

    const xSerie = uPlot.series[0];
    const xValue =
      typeof xSerie.value === 'function'
        ? xSerie.value(uPlot, uPlot.data[0][idx], 0, idx)
        : uPlot.data[0][idx];
    html += `
      <div class="${css.valueX}">
        ${header}
        ${xSerie.label} ${xValue}
      </div>`;

    let minYDist = 1000;
    const seriesValues: Array<TooltipResult | undefined> = uPlot.series
      .map((serie, i) => {
        if (serie.scale === 'x' || !serie.show) return;

        const label = yLabels[i - 1] || null;
        const valueRaw = uPlot.data[i][idx];

        const cssClass = valueRaw !== null ? css.valueY : css.valueYEmpty;
        if (isShownEmptyVal || valueRaw || valueRaw === 0) {
          const log = Math.log10(Math.abs(valueRaw || 0));
          const precision = log > -5 ? 6 - Math.max(0, Math.ceil(log)) : undefined;
          const yDist = Math.abs(uPlot.valToPos(valueRaw || 0, 'y') - (uPlot.cursor.top || 0));
          minYDist = Math.min(minYDist, yDist);

          return {
            html: `
          <div class="${cssClass}">
            <span class="${css.color}" style="background-color: ${
              seriesColors[i - 1] ?? glasbeyColor(i - 1)
            }"></span>
            ${label ? label + '<br />' : ''}
            ${serie.label}: ${valueRaw != null ? humanReadableNumber(valueRaw, precision) : 'N/A'}
          </div>`,
            val: valueRaw,
            yDist,
          };
        }
        return { val: null };
      })
      .filter((val) => val?.val !== null && !!val?.html);

    html += seriesValues
      .sort((a, b) => {
        if (!a || !b) {
          return 0;
        }
        if (a.val === null) {
          if (b.val === null) {
            return 0;
          }
          return 1;
        } else if (b.val === null) {
          return -1;
        }
        return (b.val || 0) - (a.val || 0);
      })
      .map((seriesValue) => {
        if (seriesValue?.yDist === minYDist) {
          return `<strong>${seriesValue?.html || ''}</strong>`;
        }
        return seriesValue?.html || '';
      })
      .join('');

    return html;
  };

  const _getTooltipLeftPx = (uPlot: uPlot, idx: number): number => {
    const margin = 40;
    const idxLeft = uPlot.valToPos(uPlot.data[0][idx], 'x');
    if (!tooltipEl) return idxLeft;

    const chartWidth = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().width;
    const tooltipWidth = tooltipEl.getBoundingClientRect().width;

    // right
    if (chartWidth && idxLeft + tooltipWidth >= chartWidth * 0.9) {
      return idxLeft - tooltipWidth - margin;
    }

    // left
    return idxLeft + margin;
  };

  const _updateTooltipVerticalPosition = (uPlot: uPlot, cursorTop: number) => {
    if (!tooltipEl) return;

    const chartHeight = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().height;

    const vPos = chartHeight && cursorTop > chartHeight / 2 ? 'top' : 'bottom';

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
      init: (uPlot: uPlot) => {
        uPlot.over.onmouseenter = (e) => {
          const originElement = e.relatedTarget as Element;
          if (!originElement?.className?.includes('tooltip')) {
            hide();
          }
        };
        if (closeOnMouseExit) {
          uPlot.over.onmouseout = () => {
            hide();
          };
        }
      },
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

        if (!left || left < 0 || !top || top < 0 || idx == null) {
          if (displayedIdx) hide();
          return;
        }

        const seriesWithXValue = uPlot.series.find(
          (serie, serieId) => serie.scale !== 'x' && serie.show && uPlot.data[serieId][idx] != null,
        );
        if (seriesWithXValue) {
          showIdx(uPlot, idx);
        } else {
          hide();
        }

        if (displayedIdx) {
          _updateTooltipVerticalPosition(uPlot, top);
        }
      },
    },
  };
};
