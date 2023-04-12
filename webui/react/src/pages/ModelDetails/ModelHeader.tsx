import { LeftOutlined } from '@ant-design/icons';
import { Alert, Dropdown, Space, Typography } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useMemo } from 'react';

import DeleteModelModal from 'components/DeleteModelModal';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Avatar from 'components/kit/UserAvatar';
import Link from 'components/Link';
import ModelEditModal from 'components/ModelEditModal';
import ModelMoveModal from 'components/ModelMoveModal';
import TimeAgo from 'components/TimeAgo';
import usePermissions from 'hooks/usePermissions';
import { WorkspaceDetailsTab } from 'pages/WorkspaceDetails';
import { paths } from 'routes/utils';
import Spinner from 'shared/components/Spinner';
import { ValueOf } from 'shared/types';
import { formatDatetime } from 'shared/utils/datetime';
import userStore from 'stores/users';
import { ModelItem, Workspace } from 'types';
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

const ModelHeader: React.FC<Props> = ({
  model,
  workspace,
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

  const menu: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      DeleteModel: 'delete-model',
      EditModelName: 'edit-model-name',
      MoveModel: 'move-model',
      SwitchArchived: 'switch-archive',
    } as const;

    const funcs = {
      [MenuKey.SwitchArchived]: () => {
        onSwitchArchive();
      },
      [MenuKey.EditModelName]: () => {
        modelEditModal.open();
      },
      [MenuKey.MoveModel]: () => {
        modelMoveModal.open();
      },
      [MenuKey.DeleteModel]: () => {
        deleteModelModal.open();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
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

    return { items: menuItems, onClick: onItemClick };
  }, [
    model.archived,
    canModifyModelFlag,
    canDeleteModelFlag,
    onSwitchArchive,
    modelEditModal,
    modelMoveModal,
    deleteModelModal,
  ]);

  return (
    <header className={css.base}>
      <div className={css.breadcrumbs}>
        <Breadcrumb separator="">
          <Breadcrumb.Item>
            <Link path={paths.modelList()}>
              <LeftOutlined className={css.leftIcon} />
            </Link>
          </Breadcrumb.Item>
          {workspace && (
            <Breadcrumb.Item>
              <Link
                path={
                  workspace.id === 1
                    ? paths.projectDetails(1)
                    : paths.workspaceDetails(workspace.id)
                }>
                {workspace.name}
              </Link>
            </Breadcrumb.Item>
          )}
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            <Link
              path={
                workspace?.id
                  ? paths.workspaceDetails(workspace.id, WorkspaceDetailsTab.ModelRegistry)
                  : paths.modelList()
              }>
              Model Registry
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            {model.name} ({model.id})
          </Breadcrumb.Item>
        </Breadcrumb>
      </div>
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
            <Icon name="model" size="big" />
            <h1 className={css.name}>{model.name}</h1>
          </Space>
          <Space size="small">
            <Dropdown
              disabled={!canDeleteModelFlag && !canModifyModelFlag}
              menu={menu}
              trigger={['click']}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
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
