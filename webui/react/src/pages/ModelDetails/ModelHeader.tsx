import dayjs from 'dayjs';
import Alert from 'hew/Alert';
import Button from 'hew/Button';
import Column from 'hew/Column';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Glossary, { InfoRow } from 'hew/Glossary';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Nameplate from 'hew/Nameplate';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import Tags, { tagsActionHelper } from 'hew/Tags';
import { Title } from 'hew/Typography';
import React, { useCallback, useMemo } from 'react';

import DeleteModelModal from 'components/DeleteModelModal';
import ModelEditModal from 'components/ModelEditModal';
import ModelMoveModal from 'components/ModelMoveModal';
import TimeAgo from 'components/TimeAgo';
import Avatar from 'components/UserAvatar';
import usePermissions from 'hooks/usePermissions';
import userStore from 'stores/users';
import { ModelItem, Workspace } from 'types';
import { useObservable } from 'utils/observable';
import { getDisplayName } from 'utils/user';

import css from './ModelHeader.module.scss';

interface Props {
  model: ModelItem;
  fetchModel: () => Promise<void>;
  onSwitchArchive: () => void;
  onUpdateTags: (newTags: string[]) => Promise<void>;
  workspace?: Workspace;
}

const MenuKey = {
  DeleteModel: 'delete-model',
  EditModelName: 'edit-model-name',
  MoveModel: 'move-model',
  SwitchArchived: 'switch-archive',
} as const;

const ModelHeader: React.FC<Props> = ({
  model,
  fetchModel,
  onSwitchArchive,
  onUpdateTags,
}: Props) => {
  const loadableUsers = useObservable(userStore.getUsers());
  const deleteModelModal = useModal(DeleteModelModal);
  const modelMoveModal = useModal(ModelMoveModal);
  const modelEditModal = useModal(ModelEditModal);
  const { canDeleteModel, canModifyModel } = usePermissions();
  const canDeleteModelFlag = canDeleteModel({ model });
  const canModifyModelFlag = canModifyModel({ model });

  const infoRows: InfoRow[] = useMemo(() => {
    return [
      {
        label: 'Created by',
        value: (
          <Spinner data={loadableUsers}>
            {(users) => {
              const user = users.find((user) => user.id === model.userId);
              return (
                <Row>
                  <Nameplate
                    alias={getDisplayName(user)}
                    compact
                    icon={<Avatar user={user} />}
                    name={user?.username ?? 'Unavailable'}
                  />{' '}
                  on {dayjs.utc(model.creationTime).format('MMM D, YYYY')}
                </Row>
              );
            }}
          </Spinner>
        ),
      },
      { label: 'Updated', value: <TimeAgo datetime={new Date(model.lastUpdatedTime)} /> },
      {
        label: 'Description',
        value: <div>{model.description || <span>N/A</span>}</div>,
      },
      {
        label: 'Tags',
        value: (
          <Tags
            disabled={model.archived || !canModifyModelFlag}
            ghost={false}
            tags={model.labels ?? []}
            onAction={tagsActionHelper(model.labels ?? [], onUpdateTags)}
          />
        ),
      },
    ] as InfoRow[];
  }, [canModifyModelFlag, loadableUsers, model, onUpdateTags]);

  const menu = useMemo(() => {
    const menuItems: MenuItem[] = [
      {
        disabled: model.archived || !canModifyModelFlag,
        key: MenuKey.EditModelName,
        label: 'Edit',
      },
    ];

    if (canModifyModelFlag) {
      menuItems.push({
        key: MenuKey.SwitchArchived,
        label: model.archived ? 'Unarchive' : 'Archive',
      });
      if (!model.archived) {
        menuItems.push({ key: MenuKey.MoveModel, label: 'Move' });
      }
    }
    if (canDeleteModelFlag) {
      menuItems.push({ danger: true, key: MenuKey.DeleteModel, label: 'Delete' });
    }

    return menuItems;
  }, [model.archived, canModifyModelFlag, canDeleteModelFlag]);

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.DeleteModel:
          deleteModelModal.open();
          break;
        case MenuKey.EditModelName:
          modelEditModal.open();
          break;
        case MenuKey.MoveModel:
          modelMoveModal.open();
          break;
        case MenuKey.SwitchArchived:
          onSwitchArchive();
          break;
      }
    },
    [deleteModelModal, modelEditModal, modelMoveModal, onSwitchArchive],
  );

  return (
    <header className={css.base}>
      <Column gap={16}>
        {model.archived && (
          <Alert
            message="This model has been archived and is now read-only."
            showIcon
            type="warning"
          />
        )}
        <Row justifyContent="space-between" width="fill">
          <Column>
            <Row>
              <Icon name="model" size="big" title="Model name" />
              <Title size="large">{model.name}</Title>
            </Row>
          </Column>
          <Dropdown
            disabled={!canDeleteModelFlag && !canModifyModelFlag}
            menu={menu}
            onClick={handleDropdown}>
            <Button
              icon={<Icon name="overflow-horizontal" size="small" title="Action menu" />}
              type="text"
            />
          </Dropdown>
        </Row>
        <Glossary content={infoRows} />
      </Column>
      <deleteModelModal.Component model={model} redirectOnDelete />
      <modelMoveModal.Component model={model} />
      <modelEditModal.Component fetchModel={fetchModel} model={model} />
    </header>
  );
};

export default ModelHeader;
