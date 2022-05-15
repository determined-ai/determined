import { Dropdown, Menu } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo, useState } from 'react';

import css from 'components/ActionDropdown.module.scss';
import Icon from 'components/Icon';
import { useFetchPinnedWorkspaces } from 'hooks/useFetch';
import useModalWorkspaceDelete from 'hooks/useModal/Workspace/useModalWorkspaceDelete';
import useModalWorkspaceEdit from 'hooks/useModal/Workspace/useModalWorkspaceEdit';
import { archiveWorkspace, pinWorkspace, unarchiveWorkspace, unpinWorkspace } from 'services/api';
import { DetailedUser, Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  className?: string;
  curUser?: DetailedUser;
  direction?: 'vertical' | 'horizontal';
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
  workspace: Workspace;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const WorkspaceActionDropdown: React.FC<Props> = (
  {
    workspace, children, curUser, onVisibleChange,
    className, direction = 'vertical', onComplete, trigger,
  }
  : PropsWithChildren<Props>,
) => {
  const [ canceler ] = useState(new AbortController());
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const { modalOpen: openWorkspaceDelete } = useModalWorkspaceDelete({
    onClose: onComplete,
    workspace,
  });
  const { modalOpen: openWorkspaceEdit } = useModalWorkspaceEdit({
    onClose: onComplete,
    workspace,
  });

  const userHasPermissions = useMemo(() => {
    return curUser?.isAdmin || curUser?.username === workspace.username;
  }, [ curUser?.isAdmin, curUser?.username, workspace.username ]);

  const handleArchiveClick = useCallback(async () => {
    if (workspace.archived) {
      try {
        await unarchiveWorkspace({ id: workspace.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        await archiveWorkspace({ id: workspace.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [ onComplete, workspace.archived, workspace.id ]);

  const handlePinClick = useCallback(async () => {
    if (workspace.pinned) {
      try {
        await unpinWorkspace({ id: workspace.id });
        fetchPinnedWorkspaces();
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        await pinWorkspace({ id: workspace.id });
        fetchPinnedWorkspaces();
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [ fetchPinnedWorkspaces, onComplete, workspace.id, workspace.pinned ]);

  const handleEditClick = useCallback(() => {
    openWorkspaceEdit();
  }, [ openWorkspaceEdit ]);

  const handleDeleteClick = useCallback(() => {
    openWorkspaceDelete();
  }, [ openWorkspaceDelete ]);

  const WorkspaceActionMenu = useMemo(() => {
    return (
      <Menu>
        <Menu.Item key="switchPin" onClick={handlePinClick}>
          {workspace.pinned ? 'Unpin from sidebar' : 'Pin to sidebar'}
        </Menu.Item>
        {(userHasPermissions && !workspace.archived) && (
          <Menu.Item key="edit" onClick={handleEditClick}>
            Edit...
          </Menu.Item>
        )}
        {userHasPermissions && (
          <Menu.Item key="switchArchive" onClick={handleArchiveClick}>
            {workspace.archived ? 'Unarchive' : 'Archive'}
          </Menu.Item>
        )}
        {(userHasPermissions) && (
          <>
            <Menu.Divider />
            <Menu.Item danger key="delete" onClick={handleDeleteClick}>Delete...</Menu.Item>
          </>
        )}
      </Menu>
    );
  }, [ handlePinClick,
    workspace.pinned,
    workspace.archived,
    userHasPermissions,
    handleEditClick,
    handleArchiveClick,
    handleDeleteClick ]);

  return children ? (
    <Dropdown
      overlay={WorkspaceActionMenu}
      placement="bottomLeft"
      trigger={trigger ?? [ 'contextMenu', 'click' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown
        overlay={WorkspaceActionMenu}
        placement="bottomRight"
        trigger={trigger ?? [ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default WorkspaceActionDropdown;
