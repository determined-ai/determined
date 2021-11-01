import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import { relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { ModelItem } from 'types';
import { formatDatetime } from 'utils/date';

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
    <header style={{
      backgroundColor: 'var(--theme-colors-monochrome-17)',
      borderBottom: '1px solid var(--theme-colors-monochrome-12)',
      margin: 0,
      padding: 12,
      width: '100%',
    }}>
      <div style={{
        borderBottom: '1px solid var(--theme-colors-monochrome-12)',
        paddingBottom: 8,
      }}>
        <Breadcrumb separator="">
          <Breadcrumb.Item href="det/models">
            <LeftOutlined style={{ marginRight: 10 }} />
          </Breadcrumb.Item>
          <Breadcrumb.Item href="det/models">Model Registry</Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>{model.name}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div style={{
        marginLeft: 24,
        marginRight: 24,
      }}>
        <div style={{
          alignItems: 'center',
          display: 'flex',
          justifyContent: 'space-between',
        }}>
          <div>
            <img />
            <h1>{model.name}</h1>
          </div>
          <div style={{ display: 'flex', gap: 4 }}>
            <Dropdown overlay={(
              <Menu>
                <Menu.Item key="add-metadata" onClick={onAddMetadata}>Add Metadata</Menu.Item>
                <Menu.Item key="switch-archive" onClick={onSwitchArchive}>
                  {archived ? 'Unarchive' : 'Archive'}
                </Menu.Item>
                <Menu.Item danger key="delete-model" onClick={() => showConfirmDelete(model)}>
                  Delete
                </Menu.Item>
              </Menu>
            )}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </div>
        </div>
        <InfoBox rows={infoRows} seperator={false} />
      </div>
    </header>
  );
};

export default ModelHeader;
