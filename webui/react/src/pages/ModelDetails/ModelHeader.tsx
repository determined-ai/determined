import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import { relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { ModelItem } from 'types';
import { formatDatetime } from 'utils/date';

import css from './ModelHeader.module.scss';

interface Props {
  archived: boolean;
  model: ModelItem;
  onAddMetadata: () => void;
  onDelete: () => void;
  onSaveDescription: (editedDescription: string) => Promise<void>
  onSwitchArchive: () => void;
}

const ModelHeader: React.FC<Props> = (
  { model, archived, onAddMetadata, onDelete, onSwitchArchive, onSaveDescription }: Props,
) => {
  const { auth: { user } } = useStore();

  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content:
      (<Space>
        {userRenderer(model.username, model, 0)}
        {model.username + ' on ' +
      formatDatetime(model.creationTime, 'MMM D, YYYY', false)}
      </Space>),
      label: 'Created by',
    },
    { content: relativeTimeRenderer(new Date(model.lastUpdatedTime)), label: 'Updated' },
    {
      content: <InlineEditor
        placeholder="Add description..."
        value={model.description ?? ''}
        onSave={onSaveDescription} />,
      label: 'Description',
    },
    {
      content: <TagList
        ghost={false}
        tags={model.labels ?? []}
      />,
      label: 'Tags',
    } ] as InfoRow[];
  }, [ model, onSaveDescription ]);

  const isDeletable = user?.isAdmin || user?.username === model.username;

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
              <LeftOutlined style={{ marginRight: 10 }} />
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link path={paths.modelList()}>
              Model Registry
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>{model.name}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div className={css.headerContent}>
        <div className={css.mainRow}>
          <div>
            <img />
            <h1>{model.name}</h1>
          </div>
          <Space size="small">
            <Dropdown overlay={(
              <Menu>
                <Menu.Item key="add-metadata" onClick={onAddMetadata}>Add Metadata</Menu.Item>
                <Menu.Item key="switch-archive" onClick={onSwitchArchive}>
                  {archived ? 'Unarchive' : 'Archive'}
                </Menu.Item>
                <Menu.Item
                  danger
                  disabled={!isDeletable}
                  key="delete-model"
                  onClick={() => showConfirmDelete(model)}>
                  Delete
                </Menu.Item>
              </Menu>
            )}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </Space>
        </div>
        <InfoBox rows={infoRows} seperator={false} />
      </div>
    </header>
  );
};

export default ModelHeader;
