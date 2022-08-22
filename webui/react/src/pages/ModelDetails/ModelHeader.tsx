import { LeftOutlined } from '@ant-design/icons';
import { Alert, Breadcrumb, Button, Dropdown, Menu, Space } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import { relativeTimeRenderer } from 'components/Table';
import TagList from 'components/TagList';
import Avatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import useModalModelDelete from 'hooks/useModal/Model/useModalModelDelete';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { formatDatetime } from 'shared/utils/datetime';
import { ModelItem } from 'types';
import { getDisplayName } from 'utils/user';

import css from './ModelHeader.module.scss';

interface Props {
  model: ModelItem;
  onSaveDescription: (editedDescription: string) => Promise<void>
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
  const { users, auth: { user } } = useStore();

  const { contextHolder, modalOpen } = useModalModelDelete();

  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content: (
        <Space>
          <Avatar userId={model.userId} />
          {`${getDisplayName(users.find((user) => user.id === model.userId))} on
          ${formatDatetime(model.creationTime, { format: 'MMM D, YYYY' })}`}
        </Space>
      ),
      label: 'Created by',
    },
    { content: relativeTimeRenderer(new Date(model.lastUpdatedTime)), label: 'Updated' },
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
    } ] as InfoRow[];
  }, [ model, onSaveDescription, onUpdateTags, users ]);

  const handleDelete = useCallback(() => modalOpen(model), [ modalOpen, model ]);

  const menu = useMemo(() => {
    enum MenuKey {
      SWITCH_ARCHIVED = 'switch-archive',
      DELETE_MODEL = 'delete-model',
    }

    const funcs = {
      [MenuKey.SWITCH_ARCHIVED]: () => { onSwitchArchive(); },
      [MenuKey.DELETE_MODEL]: () => { handleDelete(); },
    };

    const onItemClick:MenuProps['onClick'] = (e) => {
      funcs[e.key as MenuKey]();
    };

    const menuItems: MenuProps['items'] = [
      { key: MenuKey.SWITCH_ARCHIVED, label: model.archived ? 'Unarchive' : 'Archive' },
    ];

    if (user?.id === model.userId || user?.isAdmin) {
      menuItems.push({ danger: true, key: MenuKey.DELETE_MODEL, label: 'Delete' });
    }

    return <Menu items={menuItems} onClick={onItemClick} />;
  }, [ handleDelete, model.archived, model.userId, onSwitchArchive, user?.id, user?.isAdmin ]);

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
            <Link path={paths.modelList()}>
              Model Registry
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>{model.name} ({model.id})</Breadcrumb.Item>
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
            <Dropdown
              overlay={menu}
              trigger={[ 'click' ]}>
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
