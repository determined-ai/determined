import { LeftOutlined } from '@ant-design/icons';
import { Alert, Breadcrumb, Dropdown, Space } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Button from 'components/kit/Button';
import Link from 'components/Link';
import TagList from 'components/TagList';
import TimeAgo from 'components/TimeAgo';
import Avatar from 'components/UserAvatar';
import useModalModelDelete from 'hooks/useModal/Model/useModalModelDelete';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { formatDatetime } from 'shared/utils/datetime';
import { useUsers } from 'stores/users';
import { ModelItem } from 'types';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

import css from './ModelHeader.module.scss';

interface Props {
  model: ModelItem;
  onSaveDescription: (editedDescription: string) => Promise<void>;
  onSaveName: (editedName: string) => Promise<Error | void>;
  onSwitchArchive: () => void;
  onUpdateTags: (newTags: string[]) => Promise<void>;
}

const ModelHeader: React.FC<Props> = ({
  model,
  onSaveDescription,
  onSaveName,
  onSwitchArchive,
  onUpdateTags,
}: Props) => {
  const users = Loadable.match(useUsers(), {
    Loaded: (cUser) => cUser.users,
    NotLoaded: () => [],
  }); // TODO: handle loading state
  const { canDeleteModel } = usePermissions();
  const { contextHolder, modalOpen } = useModalModelDelete();

  const infoRows: InfoRow[] = useMemo(() => {
    const user = users.find((user) => user.id === model.userId);
    return [
      {
        content: (
          <Space>
            <Avatar user={user} />
            {`${getDisplayName(user)} on
          ${formatDatetime(model.creationTime, { format: 'MMM D, YYYY' })}`}
          </Space>
        ),
        label: 'Created by',
      },
      { content: <TimeAgo datetime={new Date(model.lastUpdatedTime)} />, label: 'Updated' },
      {
        content: (
          <InlineEditor
            disabled={model.archived}
            placeholder={model.archived ? 'Archived' : 'Add description...'}
            value={model.description ?? ''}
            onSave={onSaveDescription}
          />
        ),
        label: 'Description',
      },
      {
        content: (
          <TagList
            disabled={model.archived}
            ghost={false}
            tags={model.labels ?? []}
            onChange={onUpdateTags}
          />
        ),
        label: 'Tags',
      },
    ] as InfoRow[];
  }, [model, onSaveDescription, onUpdateTags, users]);

  const handleDelete = useCallback(() => modalOpen(model), [modalOpen, model]);

  const menu: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      DeleteModel: 'delete-model',
      SwitchArchived: 'switch-archive',
    } as const;

    const funcs = {
      [MenuKey.SwitchArchived]: () => {
        onSwitchArchive();
      },
      [MenuKey.DeleteModel]: () => {
        handleDelete();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      { key: MenuKey.SwitchArchived, label: model.archived ? 'Unarchive' : 'Archive' },
    ];

    if (canDeleteModel({ model })) {
      menuItems.push({ danger: true, key: MenuKey.DeleteModel, label: 'Delete' });
    }

    return { items: menuItems, onClick: onItemClick };
  }, [canDeleteModel, handleDelete, model, onSwitchArchive]);

  return (
    <header className={css.base}>
      <div className={css.breadcrumbs}>
        <Breadcrumb separator="">
          <Breadcrumb.Item>
            <Link path={paths.modelList()}>
              <LeftOutlined className={css.leftIcon} />
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link path={paths.modelList()}>Model Registry</Link>
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
            <h1 className={css.name}>
              <InlineEditor
                allowClear={false}
                disabled={model.archived}
                placeholder="Add name..."
                value={model.name}
                onSave={onSaveName}
              />
            </h1>
          </Space>
          <Space size="small">
            <Dropdown menu={menu} trigger={['click']}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </Space>
        </div>
        <InfoBox rows={infoRows} separator={false} />
      </div>
      {contextHolder}
    </header>
  );
};

export default ModelHeader;
