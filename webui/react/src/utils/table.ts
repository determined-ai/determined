import { GridCell, GridCellKind } from '@glideapps/glide-data-grid';

export const EMPTY_CELL: GridCell = {
  allowOverlay: false,
  data: '-',
  displayData: '-',
  kind: GridCellKind.Text,
} as const;

export const handleEmptyCell = <T>(
  param: T | undefined,
  cell: (datum: T) => GridCell,
  onlyUndefined = true,
): GridCell => {
  if ((onlyUndefined && param === undefined) || (!onlyUndefined && !param)) return EMPTY_CELL;
  return cell(param!);
};
