import { GridCell } from '@glideapps/glide-data-grid';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import Dropdown, { DropdownEvent, MenuItem } from 'hew/Dropdown';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { copyToClipboard } from 'hew/utils/functions';
import { isString } from 'lodash';
import React, { useCallback, useMemo } from 'react';

import usePermissions from 'hooks/usePermissions';
import FlatRunMoveModalComponent from 'pages/FlatRuns/FlatRunMoveModal';
import { handlePath } from 'routes/utils';
import {
  archiveRuns,
  deleteRuns,
  killRuns,
  pauseRuns,
  resumeRuns,
  unarchiveRuns,
} from 'services/api';
import { FlatRun, FlatRunAction, ValueOf } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { getActionsForFlatRun } from 'utils/flatRun';
import { capitalize } from 'utils/string';

interface Props {
  children?: React.ReactNode;
  cell?: GridCell;
  run: FlatRun;
  link?: string;
  makeOpen?: boolean;
  onComplete?: ContextMenuCompleteHandlerProps<FlatRunAction, void>;
  onLink?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  projectId: number;
}

export const Action = {
  Copy: 'Copy Value',
  NewTab: 'Open Link in New Tab',
  NewWindow: 'Open Link in New Window',
  ...FlatRunAction,
};

type Action = ValueOf<typeof Action>;

const dropdownActions = [
  Action.Archive,
  Action.Unarchive,
  Action.Kill,
  Action.Move,
  Action.Pause,
  Action.Resume,
  Action.Delete,
];

const RunActionDropdown: React.FC<Props> = ({
  run,
  cell,
  link,
  makeOpen,
  onComplete,
  onLink,
  onVisibleChange,
  projectId,
}: Props) => {
  const { Component: FlatRunMoveComponentModal, open: flatRunMoveModalOpen } =
    useModal(FlatRunMoveModalComponent);
  const confirm = useConfirm();
  const { openToast } = useToast();

  const menuItems = getActionsForFlatRun(run, dropdownActions, usePermissions()).map(
    (action: FlatRunAction) => {
      return { danger: action === Action.Delete, key: action, label: action };
    },
  );

  const cellCopyData = useMemo(() => {
    if (cell && 'displayData' in cell && isString(cell.displayData)) return cell.displayData;
    if (cell?.copyData && cell.copyData !== '-') return cell.copyData;
    return undefined;
  }, [cell]);

  const dropdownMenu = useMemo(() => {
    const items: MenuItem[] = [];
    if (link) {
      items.push(
        { key: Action.NewTab, label: Action.NewTab },
        { key: Action.NewWindow, label: Action.NewWindow },
        { type: 'divider' },
      );
    }
    if (cellCopyData) {
      items.push({ key: Action.Copy, label: Action.Copy });
    }
    items.push(...menuItems);
    return items;
  }, [link, menuItems, cellCopyData]);

  const handleDropdown = useCallback(
    async (action: string, e: DropdownEvent) => {
      try {
        switch (action) {
          case Action.NewTab:
            handlePath(e, { path: link, popout: 'tab' });
            await onLink?.();
            break;
          case Action.NewWindow:
            handlePath(e, { path: link, popout: 'window' });
            await onLink?.();
            break;
          case Action.Archive:
            await archiveRuns({ projectId, runIds: [run.id] });
            await onComplete?.(action, run.id);
            break;
          case Action.Kill:
            confirm({
              content: `Are you sure you want to kill run ${run.id}?`,
              danger: true,
              okText: 'Kill',
              onConfirm: async () => {
                await killRuns({ projectId, runIds: [run.id] });
                await onComplete?.(action, run.id);
              },
              onError: handleError,
              title: 'Confirm Run Kill',
            });
            break;
          case Action.Unarchive:
            await unarchiveRuns({ projectId, runIds: [run.id] });
            await onComplete?.(action, run.id);
            break;
          case Action.Delete:
            confirm({
              content: `Are you sure you want to delete run ${run.id}?`,
              danger: true,
              okText: 'Delete',
              onConfirm: async () => {
                await deleteRuns({ projectId, runIds: [run.id] });
                await onComplete?.(action, run.id);
              },
              onError: handleError,
              title: 'Confirm Run Deletion',
            });
            break;
          case Action.Move:
            flatRunMoveModalOpen();
            break;
          case Action.Pause:
            confirm({
              content: `Are you sure you want to pause run ${run.id}?`,
              okText: 'Pause',
              onConfirm: async () => {
                await pauseRuns({ projectId, runIds: [run.id] });
                await onComplete?.(action, run.id);
              },
              onError: handleError,
              title: 'Confirm Run Pause',
            });
            break;
          case Action.Resume:
            confirm({
              content: `Are you sure you want to resume run ${run.id}?`,
              okText: 'Resume',
              onConfirm: async () => {
                await resumeRuns({ projectId, runIds: [run.id] });
                await onComplete?.(action, run.id);
              },
              onError: handleError,
              title: 'Confirm Run Resume',
            });
            break;
          case Action.Copy:
            await copyToClipboard(cellCopyData ?? '');
            openToast({
              severity: 'Confirm',
              title: 'Value has been copied to clipboard.',
            });
            break;
        }
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: `Unable to ${action} run ${run.id}.`,
          publicSubject: `${capitalize(action)} failed.`,
          silent: false,
          type: ErrorType.Server,
        });
      } finally {
        onVisibleChange?.(false);
      }
    },
    [
      link,
      onLink,
      run.id,
      onComplete,
      confirm,
      cellCopyData,
      openToast,
      onVisibleChange,
      projectId,
      flatRunMoveModalOpen,
    ],
  );

  const shared = (
    <FlatRunMoveComponentModal
      flatRuns={[run]}
      sourceProjectId={projectId}
      sourceWorkspaceId={run.workspaceId}
      onSubmit={() => onComplete?.(FlatRunAction.Move, run.id)}
    />
  );

  return (
    <>
      <Dropdown isContextMenu menu={dropdownMenu} open={makeOpen} onClick={handleDropdown}>
        <div />
      </Dropdown>
      {shared}
    </>
  );
};

export default RunActionDropdown;
