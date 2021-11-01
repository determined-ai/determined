import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import { relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/date';

import css from './ModelVersionHeader.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onAddMetadata: () => void;
  onDeregisterVersion: () => void;
  onDownload: () => void;
  onSaveDescription: (editedNotes: string) => Promise<void>;
}

const ModelVersionHeader: React.FC<Props> = (
  { modelVersion, onAddMetadata, onDeregisterVersion, onDownload, onSaveDescription }: Props,
) => {
  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content:
      (<Space>
        {userRenderer(modelVersion.username, modelVersion, 0)}
        {modelVersion.username + ' on ' +
      formatDatetime(modelVersion.creationTime, 'MMM D, YYYY', false)}
      </Space>),
      label: 'Created by',
    },
    {
      content: relativeTimeRenderer(
        new Date(modelVersion.lastUpdatedTime ?? modelVersion.creationTime),
      ),
      label: 'Updated',
    },
    {
      content: <InlineEditor
        placeholder="Add description..."
        value={modelVersion.comment ?? ''}
        onSave={onSaveDescription} />,
      label: 'Description',
    },
    {
      content: <TagList
        ghost={false}
        tags={modelVersion.labels ?? []}
      />,
      label: 'Tags',
    } ] as InfoRow[];
  }, [ modelVersion, onSaveDescription ]);

  const showConfirmDelete = useCallback((version: ModelVersion) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this version "Version ${version.version}" 
      from this model?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Version',
      okType: 'danger',
      onOk: () => onDeregisterVersion(),
      title: 'Confirm Delete',
    });
  }, [ onDeregisterVersion ]);

  return (
    <header className={css.base}>
      <div className={css.breadcrumbs}>
        <Breadcrumb separator="">
          <Breadcrumb.Item href={`det/models/${modelVersion.model.id}`}>
            <LeftOutlined style={{ marginRight: 10 }} />
          </Breadcrumb.Item>
          <Breadcrumb.Item href="det/models">Model Registry</Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item href={`det/models/${modelVersion.model.id}`}>
            {modelVersion.model.name}
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>Version {modelVersion.version}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div className={css.headerContent}>
        <div className={css.mainRow}>
          <div className={css.title}>
            <div className={css.versionBox}>
              V{modelVersion.version}
            </div>
            <h1 className={css.versionName}>Version {modelVersion.version}</h1>
          </div>
          <div className={css.buttons}>
            <Button onClick={onDownload}>Download Model</Button>
            <Dropdown overlay={(
              <Menu>
                <Menu.Item key="add-metadata" onClick={onAddMetadata}>Add Metadata</Menu.Item>
                <Menu.Item
                  danger
                  key="deregister-version"
                  onClick={() => showConfirmDelete(modelVersion)}>
                  Deregister Version
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

export default ModelVersionHeader;
