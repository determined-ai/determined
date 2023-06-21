import { useCallback, useMemo, useRef } from 'react';
import uPlot, { Plugin } from 'uplot';

import { humanReadableNumber } from 'utils/number';
import { generateAlphaNumeric } from 'utils/string';

import { UPlotData } from '../types';

import { X_INDEX, Y_INDEX } from './UPlotScatter.utils';
import css from './useScatterPointTooltipPlugin.module.scss';

interface Props {
  labels?: (string | null)[];
  offsetX?: number;
  offsetY?: number;
}

interface UPlotState {
  dataIndex?: number;
  previousDataIndex?: number;
  seriesIndex: number;
}

interface KeyValue {
  key: string;
  value: string;
}

const ROOT_SELECTOR = '.u-over';
const DEFAULT_OFFSET_X = 6;
const DEFAULT_OFFSET_Y = 6;

const createTooltipContent = (tooltip: HTMLDivElement, keyValues: KeyValue[]) => {
  keyValues.forEach(({ key, value }) => {
    let row = tooltip.querySelector(`[data-attr="${key}"]`);
    if (!row) {
      row = document.createElement('div');
      row.setAttribute('data-attr', key);
      row.className = css.row;

      const label = document.createElement('span');
      label.className = css.label;
      row.appendChild(label);

      const info = document.createElement('span');
      info.className = css.info;
      row.appendChild(info);

      tooltip.appendChild(row);
    }

    if (row.firstElementChild) row.firstElementChild.textContent = key;
    if (row.lastElementChild) row.lastElementChild.textContent = value;
  });
};

const useScatterPointTooltipPlugin = (props: Props = {}): Plugin => {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const tooltipId = useRef(generateAlphaNumeric());
  const tooltipRef = useRef<HTMLDivElement>();
  const tooltipVisible = useRef(false);
  const uPlotRef = useRef<UPlotState>({
    dataIndex: undefined,
    previousDataIndex: undefined,
    seriesIndex: 1,
  });

  const hideTooltip = useCallback(() => {
    if (!tooltipRef.current || !tooltipVisible.current) return;
    tooltipRef.current.className = css.tooltip;
    tooltipVisible.current = false;
    uPlotRef.current.previousDataIndex = undefined;
  }, []);

  const setTooltip = useCallback(
    (u: uPlot) => {
      if (!tooltipRef.current) return;

      const { dataIndex, seriesIndex } = uPlotRef.current;
      if (dataIndex == null || dataIndex === uPlotRef.current.previousDataIndex) return;

      const xData = u.data[seriesIndex][X_INDEX] as unknown as UPlotData[];
      const yData = u.data[seriesIndex][Y_INDEX] as unknown as UPlotData[];
      const x = xData[dataIndex];
      const y = yData[dataIndex];
      if (x == null || y == null) return;

      const keyValues = u.data[seriesIndex]
        .map((data: unknown, index: number) => {
          if (data == null) return null;

          const label = props.labels?.[index];
          if (label == null) return null;

          const value = (data as UPlotData[])[dataIndex];
          if (value == null) return null;

          return { key: label, value: humanReadableNumber(value) };
        })
        .filter((keyValue) => keyValue != null) as KeyValue[];
      createTooltipContent(tooltipRef.current, keyValues);

      /**
       * Tooltip has to be shown with the updated content
       * in order to calculate the bounding rect.
       */
      tooltipRef.current.className = [css.tooltip, css.show].join(' ');
      tooltipVisible.current = true;

      const chartRect = u.root.querySelector(ROOT_SELECTOR)?.getBoundingClientRect();
      if (!chartRect) return;

      /**
       * Calculate where the tooltip should be placed based on
       * the size of the tooltip content and the cursor position.
       */
      const tooltipRect = tooltipRef.current.getBoundingClientRect();
      const valueLeft = u.valToPos(x, u.axes[0].scale || 'x');
      const valueTop = u.valToPos(y, u.axes[1].scale || 'y');
      const isLeftHalf = valueLeft < chartRect.width / 2;
      const isTopHalf = valueTop < chartRect.height / 2;
      const left = isLeftHalf ? valueLeft : valueLeft - tooltipRect.width;
      const top = isTopHalf ? valueTop : valueTop - tooltipRect.height;
      const offsetX = (isLeftHalf ? 1 : -1) * (props.offsetX || DEFAULT_OFFSET_X);
      const offsetY = (isTopHalf ? 1 : -1) * (props.offsetY || DEFAULT_OFFSET_Y);
      tooltipRef.current.style.left = `${left + offsetX}px`;
      tooltipRef.current.style.top = `${top + offsetY}px`;

      uPlotRef.current.previousDataIndex = uPlotRef.current.dataIndex;
    },
    [props.labels, props.offsetX, props.offsetY],
  );

  const plugin = useMemo(
    () => ({
      hooks: {
        ready: (u: uPlot) => {
          const tooltip = document.getElementById(tooltipId.current);
          if (tooltip) return;

          tooltipRef.current = document.createElement('div');
          tooltipRef.current.id = tooltipId.current;
          tooltipRef.current.className = css.tooltip;

          rootRef.current = u.root.querySelector<HTMLDivElement>(ROOT_SELECTOR);
          rootRef.current?.appendChild(tooltipRef.current);
        },
        setCursor: (u: uPlot) => {
          uPlotRef.current.dataIndex = u.cursor.dataIdx?.(u, 1, 0, 0);

          if (uPlotRef.current.dataIndex != null) {
            setTooltip(u);
          } else {
            hideTooltip();
          }
        },
      },
    }),
    [hideTooltip, setTooltip],
  );

  return plugin;
};

export default useScatterPointTooltipPlugin;
