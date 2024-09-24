import { GridCell, GridCellKind } from '@glideapps/glide-data-grid';

export const EMPTY_CELL: GridCell = {
  allowOverlay: false,
  data: '-',
  displayData: '-',
  kind: GridCellKind.Text,
} as const;

export const handleEmptyCell = <T>(
  uncertainData: T | undefined,
  cell: (data: T) => GridCell,
  allowFalsy = true, // if allowFalsy === false, then falsy values such as 0 or the empty string will be treated as empty
): GridCell => {
  if (uncertainData === undefined || uncertainData === null) return EMPTY_CELL;
  if (allowFalsy === false && !uncertainData) return EMPTY_CELL;
  return cell(uncertainData);
};
