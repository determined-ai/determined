import { PushpinOutlined } from '@ant-design/icons';
import { Typography } from 'antd';
import { useObservable } from 'micro-observables';
import React from 'react';

import DynamicIcon from 'components/DynamicIcon';
import Card from 'components/kit/Card';
import Avatar from 'components/kit/UserAvatar';
import { paths } from 'routes/utils';
import { pluralizer } from 'shared/utils/string';
import usersStore from 'stores/users';
import { Workspace } from 'types';
import { Loadable } from 'utils/loadable';

import { useWorkspaceActionMenu } from './WorkspaceActionDropdown';
import css from './WorkspaceCard.module.scss';

interface Props {
  fetchWorkspaces?: () => void;
  workspace: Workspace;
}

const WorkspaceCard: React.FC<Props> = ({ workspace, fetchWorkspaces }: Props) => {
  const { menuProps, contextHolders } = useWorkspaceActionMenu({
    onComplete: fetchWorkspaces,
    workspace,
  });

  const loadableUser = useObservable(usersStore.getUser(workspace.userId));
  const user = Loadable.match(loadableUser, {
    Loaded: (user) => user,
    NotLoaded: () => undefined,
  }); // TODO: handle loading state

  const classnames = [css.base];
  if (workspace.archived) classnames.push(css.archived);

  return (
    <Card
      actionMenu={!workspace.immutable ? menuProps : undefined}
      href={paths.workspaceDetails(workspace.id)}
      size="medium">
      <div className={classnames.join(' ')}>
        <div className={css.icon}>
          <DynamicIcon name={workspace.name} size={78} />
        </div>
        <div className={css.info}>
          <div className={css.nameRow}>
            <Typography.Title className={css.name} ellipsis={{ rows: 1, tooltip: true }} level={5}>
              {workspace.name}
            </Typography.Title>
            {workspace.pinned && <PushpinOutlined className={css.pinned} />}
          </div>
          <p className={css.projects}>
            {workspace.numProjects} {pluralizer(workspace.numProjects, 'project')}
          </p>
          <div className={css.avatarRow}>
            <div className={css.avatar}>
              <Avatar user={user} />
            </div>
            {workspace.archived && <div className={css.archivedBadge}>Archived</div>}
          </div>
        </div>
      </div>
      {contextHolders}
    </Card>
  );
};

export default WorkspaceCard;
