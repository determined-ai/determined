import { GridCell, GridCellKind } from '@glideapps/glide-data-grid';
import { EMPTY_CELL, handleEmptyCell } from './table';
import fc from 'fast-check';

const generateGridCell = (value: unknown): GridCell => {
  return {
    kind: GridCellKind.Text,
    data: String(value),
    displayData: String(value),
    allowOverlay: false,
  };
};

describe('Table Utilities', () => {
  describe('handleEmptyCell', () => {
    it('should return passed cell for any string value', () => {
      fc.assert(
        fc.property(fc.string(), (value) => {
          expect(handleEmptyCell(value, (data) => generateGridCell(data))).toEqual(
            generateGridCell(value),
          );
        }),
      );
    });
    it('should return EMPTY_CELL for undefined value', () => {
      const value = undefined;
      expect(handleEmptyCell(value, (data) => generateGridCell(data))).not.toEqual(
        generateGridCell(value),
      );
      expect(handleEmptyCell(value, (data) => generateGridCell(data))).toEqual(EMPTY_CELL);
    });
    it('should return EMPTY_CELL for null value', () => {
      const value = null;
      expect(handleEmptyCell(value, (data) => generateGridCell(data))).not.toEqual(
        generateGridCell(value),
      );
      expect(handleEmptyCell(value, (data) => generateGridCell(data))).toEqual(EMPTY_CELL);
    });
    it('should return passed cell for any non-empty string value when allowFalsy is false', () => {
      fc.assert(
        fc.property(fc.string({ minLength: 1 }), (value) => {
          expect(handleEmptyCell(value, (data) => generateGridCell(data))).toEqual(
            generateGridCell(value),
          );
        }),
      );
    });
    it('should return EMPTY_CELL for empty string value when allowFalsy is false', () => {
      const value = '';
      expect(handleEmptyCell(value, (data) => generateGridCell(data), false)).not.toEqual(
        generateGridCell(value),
      );
      expect(handleEmptyCell(value, (data) => generateGridCell(data), false)).toEqual(EMPTY_CELL);
    });
  });
});
