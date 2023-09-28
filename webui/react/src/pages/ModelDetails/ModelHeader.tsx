import { Alert, Space, Typography } from 'antd';
import React, { useCallback, useMemo } from 'react';

import DeleteModelModal from 'components/DeleteModelModal';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import ModelEditModal from 'components/ModelEditModal';
import ModelMoveModal from 'components/ModelMoveModal';
import TimeAgo from 'components/TimeAgo';
import Avatar from 'components/UserAvatar';
import usePermissions from 'hooks/usePermissions';
import userStore from 'stores/users';
import { ModelItem, Workspace } from 'types';
import { formatDatetime } from 'utils/datetime';
import { Loadable } from 'utils/loadable';
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
  const users = Loadable.getOrElse([], loadableUsers);
  const deleteModelModal = useModal(DeleteModelModal);
  const modelMoveModal = useModal(ModelMoveModal);
  const modelEditModal = useModal(ModelEditModal);
  const { canDeleteModel, canModifyModel } = usePermissions();
  const canDeleteModelFlag = canDeleteModel({ model });
  const canModifyModelFlag = canModifyModel({ model });

  const infoRows: InfoRow[] = useMemo(() => {
    const user = users.find((user) => user.id === model.userId);

    return [
      {
        content: (
          <Space>
            <Spinner conditionalRender spinning={Loadable.isLoading(loadableUsers)}>
              <>
                <Avatar user={user} />
                {`${getDisplayName(user)} on
                ${formatDatetime(model.creationTime, { format: 'MMM D, YYYY' })}`}
              </>
            </Spinner>
          </Space>
        ),
        label: 'Created by',
      },
      { content: <TimeAgo datetime={new Date(model.lastUpdatedTime)} />, label: 'Updated' },
      {
        content: (
          <div>
            {(model.description ?? '') || (
              <Typography.Text disabled={model.archived || !canModifyModelFlag}>
                N/A
              </Typography.Text>
            )}
          </div>
        ),
        label: 'Description',
      },
      {
        content: (
          <Tags
            disabled={model.archived || !canModifyModelFlag}
            ghost={false}
            tags={model.labels ?? []}
            onAction={tagsActionHelper(model.labels ?? [], onUpdateTags)}
          />
        ),
        label: 'Tags',
      },
    ] as InfoRow[];
  }, [canModifyModelFlag, loadableUsers, model, onUpdateTags, users]);

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
      {model.archived && (
        <Alert
          message="This model has been archived and is now read-only."
          showIcon
          style={{ marginTop: 8 }}
          type="warning"
        />
      )}
      <div className={css.headerContent}>
        <div className={css.mainRow}>
          <Space className={css.nameAndIcon}>
            <Icon name="model" size="big" title="Model name" />
            <h1 className={css.name}>{model.name}</h1>
          </Space>
          <Space size="small">
            <Dropdown
              disabled={!canDeleteModelFlag && !canModifyModelFlag}
              menu={menu}
              onClick={handleDropdown}>
              <Button
                icon={<Icon name="overflow-horizontal" size="small" title="Action menu" />}
                type="text"
              />
            </Dropdown>
          </Space>
        </div>
        <InfoBox rows={infoRows} separator={false} />
      </div>
      <deleteModelModal.Component model={model} />
      <modelMoveModal.Component model={model} />
      <modelEditModal.Component fetchModel={fetchModel} model={model} />
    </header>
  );
};

export default ModelHeader;
