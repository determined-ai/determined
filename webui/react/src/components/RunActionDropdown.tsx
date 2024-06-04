import { GridCell } from '@glideapps/glide-data-grid';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import Dropdown, { DropdownEvent, MenuItem } from 'hew/Dropdown';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { copyToClipboard } from 'hew/utils/functions';
import React, { MouseEvent, useCallback, useMemo } from 'react';

import usePermissions from 'hooks/usePermissions';
import FlatRunMoveModalComponent from 'pages/FlatRuns/FlatRunMoveModal';
import { handlePath } from 'routes/utils';
import { archiveRuns, deleteRuns, killRuns, unarchiveRuns } from 'services/api';
import { FlatRun, FlatRunAction, ValueOf } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { getActionsForFlatRun } from 'utils/flatRun';
import { capitalize } from 'utils/string';

import { FilterFormSetWithoutId } from './FilterForm/components/type';

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
  filterFormSetWithoutId: FilterFormSetWithoutId;
}

const Action = {
  Copy: 'Copy Value',
  NewTab: 'Open Link in New Tab',
  NewWindow: 'Open Link in New Window',
  ...FlatRunAction,
};

type Action = ValueOf<typeof Action>;

const dropdownActions = [Action.Archive, Action.Unarchive, Action.Kill, Action.Move, Action.Delete];

const RunActionDropdown: React.FC<Props> = ({
  run,
  cell,
  link,
  makeOpen,
  onComplete,
  onLink,
  onVisibleChange,
  filterFormSetWithoutId,
  projectId,
}: Props) => {
  const id = run.id;
  const { Component: FlatRunMoveComponentModal, open: flatRunMoveModalOpen } =
    useModal(FlatRunMoveModalComponent);
  const confirm = useConfirm();
  const { openToast } = useToast();

  const menuItems = getActionsForFlatRun(run, dropdownActions, usePermissions()).map(
    (action: FlatRunAction) => {
      return { danger: action === Action.Delete, key: action, label: action };
    },
  );

  const dropdownMenu = useMemo(() => {
    const items: MenuItem[] = [...menuItems];
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    if (cell && (cell.copyData || (cell as any).displayData)) {
      items.unshift({ key: Action.Copy, label: Action.Copy });
    }
    if (link) {
      items.unshift(
        { key: Action.NewTab, label: Action.NewTab },
        { key: Action.NewWindow, label: Action.NewWindow },
        { type: 'divider' },
      );
    }
    return items;
  }, [link, menuItems, cell]);

  const handleDropdown = useCallback(
    async (action: string, e: DropdownEvent) => {
      try {
        switch (action) {
          case Action.NewTab:
            handlePath(e as MouseEvent, { path: link, popout: 'tab' });
            await onLink?.();
            break;
          case Action.NewWindow:
            handlePath(e as MouseEvent, { path: link, popout: 'window' });
            await onLink?.();
            break;
          case Action.Archive:
            await archiveRuns({ projectId, runIds: [id] });
            await onComplete?.(action, id);
            break;
          case Action.Kill:
            confirm({
              content: `Are you sure you want to kill run ${id}?`,
              danger: true,
              okText: 'Kill',
              onConfirm: async () => {
                await killRuns({ projectId, runIds: [id] });
                await onComplete?.(action, id);
              },
              onError: handleError,
              title: 'Confirm Run Kill',
            });
            break;
          case Action.Unarchive:
            await unarchiveRuns({ projectId, runIds: [id] });
            await onComplete?.(action, id);
            break;
          case Action.Delete:
            confirm({
              content: `Are you sure you want to delete run ${id}?`,
              danger: true,
              okText: 'Delete',
              onConfirm: async () => {
                await deleteRuns({ projectId, runIds: [id] });
                await onComplete?.(action, id);
              },
              onError: handleError,
              title: 'Confirm Run Deletion',
            });
            break;
          case Action.Move:
            flatRunMoveModalOpen();
            break;
          case Action.Copy:
            /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
            await copyToClipboard((cell as any).displayData || cell?.copyData);
            openToast({
              severity: 'Confirm',
              title: 'Value has been copied to clipboard.',
            });
            break;
        }
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: `Unable to ${action} experiment ${id}.`,
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
      id,
      onComplete,
      confirm,
      cell,
      openToast,
      onVisibleChange,
      projectId,
      flatRunMoveModalOpen,
    ],
  );

  const shared = (
    <>
      <FlatRunMoveComponentModal
        filterFormSetWithoutId={filterFormSetWithoutId}
        flatRuns={[run]}
        sourceProjectId={projectId}
        sourceWorkspaceId={run.workspaceId}
        onActionComplete={() => onComplete?.(FlatRunAction.Move, id)}
      />
    </>
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
