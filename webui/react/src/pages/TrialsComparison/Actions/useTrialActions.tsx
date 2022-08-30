import React, { ReactNode, useCallback, useEffect, useMemo, useState } from 'react';

import TableBatch from 'components/Table/TableBatch';
import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';

import { encodeIdList } from '../api';
import { TrialsCollectionSpec, TrialsSelection } from '../Collections/collections';
import { TrialFilters, TrialSorter } from '../Collections/filters';
import { CollectionModalProps } from '../Collections/useModalCreateCollection';

import useModalTrialTag from './useModalTagTrials';
import {
  dispatchTrialAction,
  openTensorBoard,
  TrialAction,
  trialActionDefs,
  TrialsActionHandler,
} from './utils';

export interface TrialActionsInterface {
  dispatcher: ReactNode;
  modalContextHolder: React.ReactElement;
  resetSelection: () => void;
  selectAllMatching: boolean;
  selectTrial: (ids: unknown) => void;
  selectedTrials: number[];
}

interface Props {
  filters: TrialFilters
  openCreateModal: (p: CollectionModalProps) => void;
  refetch: () => void;
  sorter: TrialSorter;
}

// export const settingsConfig: SettingsConfig = {
//   applicableRoutespace: '/trials',
//   settings: [
//     {
//       defaultValue: [],
//       key: 'ids',
//       // skipUrlEncoding: true,
//       storageKey: 'selectedTrialIds',
//       type: { baseType: BaseType.Integer, isArray: true },
//     },
//   ],
//   storagePath: 'trials-selection',
// };

const useTrialActions = ({
  filters,
  sorter,
  openCreateModal,
  refetch,
}: Props): TrialActionsInterface => {
  // const { settings, updateSettings } =
  //   useSettings<{ ids: number[] }>(settingsConfig);

  // const selectedTrials = useMemo(() => settings.ids ?? [], [ settings.ids ]);

  // const setSelectedTrials = useCallback(
  //   (ids: number[]) => updateSettings({ ids }),
  //   [ updateSettings ],
  // );

  const [ selectedTrials, setSelectedTrials ] = useState<number[]>([]);
  const [ selectAllMatching, setSelectAllMatching ] = useState<boolean>(false);
  const handleChangeSelectionMode = useCallback(() => setSelectAllMatching((prev) => !prev), []);

  const selectTrial = useCallback((rowKeys) => setSelectedTrials(
    encodeIdList(rowKeys) ?? [],
  ), []);
  // const handleTableChange = useCallback((pageSize) => setPageSize(pageSize), []);

  const clearSelected = useCallback(() => {
    setSelectedTrials([]);
  }, []);

  const {
    contextHolder,
    modalOpen,
  } = useModalTrialTag({ onConfirm: refetch });

  const handleBatchAction = useCallback(async (action: string) => {
    const trials = selectAllMatching
      ? { filters, sorter } as TrialsCollectionSpec
      : { sorter: sorter, trialIds: selectedTrials } as TrialsSelection;

    const handle = async (handler: TrialsActionHandler) =>
      await dispatchTrialAction(action as TrialAction, trials, handler);

    await (
      action === TrialAction.AddTags
        ? handle(modalOpen)
        : action === TrialAction.TagAndCollect
          ? handle(openCreateModal)
          : action === TrialAction.OpenTensorBoard
            ? handle(openTensorBoard)
            : Promise.resolve()
    );
  }, [
    selectedTrials,
    modalOpen,
    selectAllMatching,
    sorter,
    filters,
    openCreateModal,
  ]);

  const dispatcher = (
    <TableBatch
      actions={Object.values(trialActionDefs)}
      selectAllMatching={selectAllMatching}
      selectedRowCount={selectedTrials.length}
      onAction={handleBatchAction}
      onChangeSelectionMode={handleChangeSelectionMode}
      onClear={clearSelected}
    />
  );

  return {
    dispatcher,
    modalContextHolder: contextHolder,
    resetSelection: clearSelected,
    selectAllMatching,
    selectedTrials,
    selectTrial,
  };
};
export default useTrialActions;
