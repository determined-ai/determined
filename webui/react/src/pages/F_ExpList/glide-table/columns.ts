import {
  DataEditorProps,
  GridCell,
  GridCellKind,
  Theme as GTheme,
  SizedGridColumn,
} from '@glideapps/glide-data-grid';
import { Theme } from 'hew/Theme';

import { RawJson } from 'types';
import { getPath, isString } from 'utils/data';
import { formatDatetime } from 'utils/datetime';

export const MIN_COLUMN_WIDTH = 40;
export const NO_PINS_WIDTH = 200;

export const MULTISELECT = 'selected';

export type ColumnDef<T> = SizedGridColumn & {
  id: string;
  isNumerical?: boolean;
  renderer: (record: T, idx: number) => GridCell;
  tooltip: (record: T) => string | undefined;
};

export type ColumnDefs<T> = Record<string, ColumnDef<T>>;

export interface HeatmapProps {
  min: number;
  max: number;
}

export function defaultTextColumn<T extends RawJson>(
  columnId: string,
  columnTitle: string,
  columnWidth?: number,
  dataPath?: string,
): ColumnDef<T> {
  return {
    id: columnId,
    renderer: (record) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: String(data ?? ''),
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
}

const getHeatmapPercentage = (min: number, max: number, value: number): number => {
  if (min >= max || value >= max) return 1;
  if (value <= min) return 0;
  return (value - min) / (max - min);
};

export const getHeatmapColor = (min: number, max: number, value: number): string => {
  const p = getHeatmapPercentage(min, max, value);
  const red = [44, 222];
  const green = [119, 66];
  const blue = [176, 91];
  return `rgb(${red[0] + (red[1] - red[0]) * p}, ${green[0] + (green[1] - green[0]) * p}, ${
    blue[0] + (blue[1] - blue[0]) * p
  })`;
};

export function defaultNumberColumn<T extends RawJson>(
  columnId: string,
  columnTitle: string,
  columnWidth?: number,
  dataPath?: string,
  heatmapProps?: HeatmapProps,
): ColumnDef<T> {
  return {
    id: columnId,
    renderer: (record) => {
      const data = isString(dataPath) ? getPath<number>(record, dataPath) : undefined;
      let theme: Partial<GTheme> = {};
      if (heatmapProps && data !== undefined) {
        const { min, max } = heatmapProps;
        theme = {
          accentLight: getHeatmapColor(min, max, data),
          bgCell: getHeatmapColor(min, max, data),
          textDark: 'white',
        };
      }
      return {
        allowOverlay: false,
        copyData: data !== undefined ? String(data) : '',
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
        themeOverride: theme,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
}

export function defaultDateColumn<T extends RawJson>(
  columnId: string,
  columnTitle: string,
  columnWidth?: number,
  dataPath?: string,
): ColumnDef<T> {
  return {
    id: columnId,
    renderer: (record) => {
      const data = isString(dataPath) ? getPath<string>(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: formatDatetime(String(data), { outputUTC: false }),
        data: { kind: 'text-cell' },
        kind: GridCellKind.Custom,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? columnWidthsFallback,
  };
}

export const columnWidthsFallback = 140;

// TODO: use theme here
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const getHeaderIcons = (_appTheme: Theme): DataEditorProps['headerIcons'] => ({
  allSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="5.25" y1="6.5" x2="6.75" y2="8" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="6.75" y1="8" x2="9.25" y2="5.5" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  `,
  noneSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
    </svg>
  `,
  someSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="3" y1="7" x2="11" y2="7" stroke="#929292" stroke-width="2"/>
    </svg>
  `,
});
