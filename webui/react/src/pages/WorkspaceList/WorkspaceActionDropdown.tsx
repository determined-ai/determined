import { Dropdown, Menu } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import css from 'components/ActionDropdown.module.scss';
import Icon from 'components/Icon';
import useModalWorkspaceDelete from 'hooks/useModal/Workspace/useModalWorkspaceDelete';
import { archiveWorkspace, unarchiveWorkspace } from 'services/api';
import { DetailedUser, Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  className?: string;
  curUser?: DetailedUser;
  direction?: 'vertical' | 'horizontal';
  fetchWorkspaces?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  workspace: Workspace;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const WorkspaceActionDropdown: React.FC<Props> = (
  {
    workspace, children, curUser, onVisibleChange,
    className, direction = 'vertical', fetchWorkspaces,
  }
  : PropsWithChildren<Props>,
) => {
  const { modalOpen: openWorkspaceDelete } = useModalWorkspaceDelete({
    onClose: fetchWorkspaces,
    workspace,
  });

  const userHasPermissions = useMemo(() => {
    return curUser?.isAdmin || curUser?.username === workspace.username;
  }, [ curUser?.isAdmin, curUser?.username, workspace.username ]);

  const handleArchiveClick = useCallback(() => {
    if (workspace.archived) {
      try {
        unarchiveWorkspace({ id: workspace.id });
        fetchWorkspaces?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        archiveWorkspace({ id: workspace.id });
        fetchWorkspaces?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [ fetchWorkspaces, workspace.archived, workspace.id ]);

  const handleDeleteClick = useCallback(() => {
    openWorkspaceDelete();
  }, [ openWorkspaceDelete ]);

  const WorkspaceActionMenu = useMemo(() => {
    return (
      <Menu>
        {userHasPermissions && (
          <Menu.Item key="switchArchive" onClick={handleArchiveClick}>
            {workspace.archived ? 'Unarchive' : 'Archive'}
          </Menu.Item>
        )}
        {userHasPermissions && !workspace.archived &&
        <Menu.Item danger key="delete" onClick={handleDeleteClick}>Delete...</Menu.Item>}
      </Menu>
    );
  }, [ handleArchiveClick, handleDeleteClick, workspace.archived, userHasPermissions ]);

  if (!userHasPermissions) {
    return (children as JSX.Element) ?? (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name={`overflow-${direction}`} />
        </button>
      </div>
    );
  }

  return children ? (
    <Dropdown
      overlay={WorkspaceActionMenu}
      placement="bottomLeft"
      trigger={[ 'contextMenu' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown overlay={WorkspaceActionMenu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default WorkspaceActionDropdown;
