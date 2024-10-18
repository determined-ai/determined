import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
import {
  HandleSelectionChangeType,
  RangelessSelectionType,
  SelectionType,
} from 'hew/DataGrid/DataGrid';
import { Loadable } from 'hew/utils/loadable';
import * as t from 'io-ts';
import { useCallback, useMemo } from 'react';

import { RegularSelectionType, SelectionType as SelectionState } from 'types';

export const DEFAULT_SELECTION: t.TypeOf<typeof RegularSelectionType> = {
  selections: [],
  type: 'ONLY_IN',
};

interface HasId {
  id: number;
}

interface SelectionConfig<T> {
  records: Loadable<T>[];
  selection: SelectionState;
  total: Loadable<number>;
  updateSettings: (p: Record<string, unknown>) => void;
}

interface UseSelectionReturn<T> {
  selectionSize: number;
  dataGridSelection: GridSelection;
  handleSelectionChange: HandleSelectionChangeType;
  rowRangeToIds: (range: [number, number]) => number[];
  loadedSelectedRecords: T[];
  isRangeSelected: (range: [number, number]) => boolean;
}

const useSelection = <T extends HasId>(config: SelectionConfig<T>): UseSelectionReturn<T> => {
  const loadedRecordIdMap = useMemo(() => {
    const recordMap = new Map<number, { record: T; index: number }>();

    config.records.forEach((r, index) => {
      Loadable.forEach(r, (record) => {
        recordMap.set(record.id, { index, record });
      });
    });
    return recordMap;
  }, [config.records]);

  const selectedRecordIdSet = useMemo(() => {
    if (config.selection.type === 'ONLY_IN') {
      return new Set(config.selection.selections);
    } else if (config.selection.type === 'ALL_EXCEPT') {
      const excludedSet = new Set(config.selection.exclusions);
      return new Set(
        Loadable.filterNotLoaded(config.records, (record) => !excludedSet.has(record.id)).map(
          (record) => record.id,
        ),
      );
    }
    return new Set<number>(); // should never be reached
  }, [config.records, config.selection]);

  const dataGridSelection = useMemo<GridSelection>(() => {
    let rows = CompactSelection.empty();
    if (config.selection.type === 'ONLY_IN') {
      config.selection.selections.forEach((id) => {
        const incIndex = loadedRecordIdMap.get(id)?.index;
        if (incIndex !== undefined) {
          rows = rows.add(incIndex);
        }
      });
    } else if (config.selection.type === 'ALL_EXCEPT') {
      rows = rows.add([0, config.total.getOrElse(1) - 1]);
      console.log(config.selection.exclusions);
      console.log(loadedRecordIdMap);
      config.selection.exclusions.forEach((exc) => {
        const excIndex = loadedRecordIdMap.get(exc)?.index;
        console.log(excIndex);
        if (excIndex !== undefined) {
          rows = rows.remove(excIndex);
        }
      });
    }
    console.log(rows);
    return {
      columns: CompactSelection.empty(),
      rows,
    };
  }, [loadedRecordIdMap, config.selection, config.total]);

  const loadedSelectedRecords: T[] = useMemo(() => {
    return Loadable.filterNotLoaded(config.records, (record) => selectedRecordIdSet.has(record.id));
  }, [config.records, selectedRecordIdSet]);

  const selectionSize = useMemo(() => {
    if (config.selection.type === 'ONLY_IN') {
      return config.selection.selections.length;
    } else if (config.selection.type === 'ALL_EXCEPT') {
      return config.total.getOrElse(0) - config.selection.exclusions.length;
    }
    return 0;
  }, [config.selection, config.total]);

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      const slice = config.records.slice(range[0], range[1]);
      return Loadable.filterNotLoaded(slice).map((run) => run.id);
    },
    [config.records],
  );

  const handleSelectionChange: HandleSelectionChangeType = useCallback(
    (selectionType: SelectionType | RangelessSelectionType, range?: [number, number]) => {
      let newSettings: SelectionState = { ...config.selection };

      switch (selectionType) {
        case 'add':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.delete(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.add(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'add-all':
          newSettings = {
            exclusions: [],
            type: 'ALL_EXCEPT',
          };

          break;
        case 'remove':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.add(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.delete(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'remove-all':
          newSettings = DEFAULT_SELECTION;

          break;
        case 'set':
          if (!range) return;
          newSettings = {
            ...DEFAULT_SELECTION,
            selections: Array.from(rowRangeToIds(range)),
          };

          break;
      }
      config.updateSettings({ selection: newSettings });
    },
    [config, rowRangeToIds],
  );

  const isRangeSelected = useCallback(
    (range: [number, number]): boolean => {
      if (config.selection.type === 'ONLY_IN') {
        const includedSet = new Set(config.selection.selections);
        return rowRangeToIds(range).every((id) => includedSet.has(id));
      } else if (config.selection.type === 'ALL_EXCEPT') {
        const excludedSet = new Set(config.selection.exclusions);
        return rowRangeToIds(range).every((id) => !excludedSet.has(id));
      }
      return false; // should never be reached
    },
    [rowRangeToIds, config.selection],
  );

  return {
    dataGridSelection,
    handleSelectionChange,
    isRangeSelected,
    loadedSelectedRecords,
    rowRangeToIds,
    selectionSize,
  };
};

export default useSelection;
