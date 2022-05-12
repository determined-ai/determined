import { LeftOutlined } from '@ant-design/icons';
import { Alert, Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Avatar from 'components/Avatar';
import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import showModalItemCannotDelete from 'components/ModalItemDelete';
import { relativeTimeRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { formatDatetime } from 'shared/utils/datetime';
import { ModelItem } from 'types';
import { getDisplayName } from 'utils/user';

import css from './ModelHeader.module.scss';

interface Props {
  model: ModelItem;
  onDelete: () => void;
  onSaveDescription: (editedDescription: string) => Promise<void>
  onSaveName: (editedName: string) => Promise<void>;
  onSwitchArchive: () => void;
  onUpdateTags: (newTags: string[]) => Promise<void>;
}

const ModelHeader: React.FC<Props> = (
  {
    model, onDelete, onSwitchArchive,
    onSaveDescription, onUpdateTags, onSaveName,
  }: Props,
) => {
  const { auth: { user }, users } = useStore();

  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content: (
        <Space>
          <Avatar userId={model.userId} />
          {`${getDisplayName(users.find(user => user.id === model.userId))} on 
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
          placeholder="Add description..."
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

  const isDeletable = user?.isAdmin || user?.id === model.userId;

  const showConfirmDelete = useCallback((model: ModelItem) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this model "${model.name}" and all 
      of its versions from the model registry?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Model',
      okType: 'danger',
      onOk: () => onDelete(),
      title: 'Confirm Delete',
    });
  }, [ onDelete ]);

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
              overlay={(
                <Menu>
                  <Menu.Item key="switch-archive" onClick={onSwitchArchive}>
                    {model.archived ? 'Unarchive' : 'Archive'}
                  </Menu.Item>
                  <Menu.Item
                    danger
                    key="delete-model"
                    onClick={() => isDeletable ?
                      showConfirmDelete(model) : showModalItemCannotDelete()}>
                    Delete
                  </Menu.Item>
                </Menu>
              )}
              trigger={[ 'click' ]}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </Space>
        </div>
        <InfoBox rows={infoRows} separator={false} />
      </div>
    </header>
  );
};

export default ModelHeader;
