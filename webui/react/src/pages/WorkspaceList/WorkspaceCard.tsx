import Avatar, { Size } from 'hew/Avatar';
import Badge from 'hew/Badge';
import Card from 'hew/Card';
import Column from 'hew/Column';
import Icon from 'hew/Icon';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import { Label, Title, TypographySize } from 'hew/Typography';
import { Loadable } from 'hew/utils/loadable';
import React from 'react';

import UserAvatar from 'components/UserAvatar';
import { handlePath, paths } from 'routes/utils';
import userStore from 'stores/users';
import { Workspace } from 'types';
import { useObservable } from 'utils/observable';
import { AnyMouseEvent } from 'utils/routes';
import { pluralizer } from 'utils/string';

import { useWorkspaceActionMenu } from './WorkspaceActionDropdown';
import css from './WorkspaceCard.module.scss';

interface Props {
  fetchWorkspaces?: () => void;
  workspace: Workspace;
}

const WorkspaceCard: React.FC<Props> = ({ workspace, fetchWorkspaces }: Props) => {
  const { contextHolders, menu, onClick } = useWorkspaceActionMenu({
    onComplete: fetchWorkspaces,
    workspace,
  });
  const loadableUser = useObservable(userStore.getUser(workspace.userId));
  const user = Loadable.getOrElse(undefined, loadableUser);
  const testId = `card-${workspace.name}`;

  return (
    <Card
      actionMenu={!workspace.immutable ? menu : undefined}
      size="small"
      testId={testId}
      onClick={(e: AnyMouseEvent) => handlePath(e, { path: paths.workspaceDetails(workspace.id) })}
      onDropdown={onClick}>
      <div className={workspace.archived ? css.archived : ''}>
        <Row>
          <Column width="hug">
            <Avatar palette="muted" size={Size.ExtraLarge} square text={workspace.name} />
          </Column>
          <div className={css.info}>
            <Column width={225}>
              <Row justifyContent="space-between" width="fill">
                <Title size={TypographySize.XS} truncate={{ rows: 1, tooltip: true }}>
                  {workspace.name}
                </Title>
                {workspace.pinned && <Icon name="pin" title="Pinned" />}
              </Row>
              <Row>
                <Label size="small">
                  {workspace.numProjects} {pluralizer(workspace.numProjects, 'project')}
                </Label>
              </Row>
              <Row justifyContent="space-between" width="fill">
                <Spinner conditionalRender spinning={Loadable.isNotLoaded(loadableUser)}>
                  {Loadable.isLoaded(loadableUser) && <UserAvatar user={user} />}
                </Spinner>
                {workspace.archived && (
                  <Badge
                    backgroundColor={{ h: 0, l: 40, s: 0 }}
                    data-testid="archived"
                    text="Archived"
                  />
                )}
              </Row>
            </Column>
          </div>
        </Row>
      </div>
      {contextHolders}
    </Card>
  );
};

export default WorkspaceCard;
