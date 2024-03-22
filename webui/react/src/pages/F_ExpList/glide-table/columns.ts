import {
  CompactSelection,
  GridCell,
  GridCellKind,
  Theme as GTheme,
  SizedGridColumn,
} from '@glideapps/glide-data-grid';
import _ from 'lodash';

import { RawJson } from 'types';
import { formatDatetime } from 'utils/datetime';

import { CHECKBOX_CELL } from './custom-renderers/cells/checkboxCell';
import { TEXT_CELL } from './custom-renderers/cells/textCell';

export const DEFAULT_COLUMN_WIDTH = 140;
export const MIN_COLUMN_WIDTH = 40;

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
      const data = dataPath !== undefined ? _.get(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: String(data ?? ''),
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? DEFAULT_COLUMN_WIDTH,
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
      const data = dataPath !== undefined ? _.get(record, dataPath) : undefined;
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
        copyData: String(data ?? ''),
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
        themeOverride: theme,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? DEFAULT_COLUMN_WIDTH,
  };
}

export function defaultSelectionColumn<T>(
  rowSelection: CompactSelection,
  selectAll: boolean,
): ColumnDef<T> {
  return {
    icon: selectAll ? 'allSelected' : rowSelection.length ? 'someSelected' : 'noneSelected',
    id: MULTISELECT,
    renderer: (_, idx) => ({
      allowOverlay: false,
      contentAlign: 'left',
      copyData: String(rowSelection.hasIndex(idx)),
      data: {
        checked: rowSelection.hasIndex(idx),
        kind: CHECKBOX_CELL,
      },
      kind: GridCellKind.Custom,
    }),
    themeOverride: { cellHorizontalPadding: 10 },
    title: '',
    tooltip: () => undefined,
    width: 40,
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
      const data = dataPath !== undefined ? _.get(record, dataPath) : undefined;
      return {
        allowOverlay: false,
        copyData: data ? formatDatetime(String(data), { outputUTC: false }) : '',
        data: { kind: TEXT_CELL },
        kind: GridCellKind.Custom,
      };
    },
    title: columnTitle,
    tooltip: () => undefined,
    width: columnWidth ?? DEFAULT_COLUMN_WIDTH,
  };
}
