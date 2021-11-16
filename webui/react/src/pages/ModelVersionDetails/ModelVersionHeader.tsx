import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo } from 'react';

import DownloadModelPopover from 'components/DownloadModelPopover';
import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import { relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/date';

import css from './ModelVersionHeader.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onAddMetadata: () => void;
  onDeregisterVersion: () => void;
  onSaveDescription: (editedNotes: string) => Promise<void>;
  onUpdateTags: (newTags: string[]) => Promise<void>;
}

const ModelVersionHeader: React.FC<Props> = (
  { modelVersion, onAddMetadata, onDeregisterVersion, onSaveDescription, onUpdateTags }: Props,
) => {
  const { auth: { user } } = useStore();

  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content:
      (<Space>
        {modelVersion.username ?
          userRenderer(modelVersion.username, modelVersion, 0) :
          userRenderer(modelVersion.model.username, modelVersion.model, 0)}
        {modelVersion.username ? modelVersion.username : modelVersion.model.username + ' on ' +
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
        onChange={onUpdateTags}
      />,
      label: 'Tags',
    } ] as InfoRow[];
  }, [ modelVersion, onSaveDescription, onUpdateTags ]);

  const isDeletable = user?.isAdmin
        || user?.username === modelVersion.model.username
        || user?.username === modelVersion.username;

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
          <Breadcrumb.Item>
            <Link path={paths.modelDetails(modelVersion.model.id)}>
              <LeftOutlined style={{ marginRight: 10 }} />
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link path={paths.modelList()}>
              Model Registry
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            <Link path={paths.modelDetails(modelVersion.model.id)}>
              {modelVersion.model.name}
            </Link>
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
            <DownloadModelPopover modelVersion={modelVersion}>
              <Button>Download Model</Button>
            </DownloadModelPopover>
            <Dropdown
              overlay={(
                <Menu>
                  {Object.keys(modelVersion.metadata ?? {}).length === 0 &&
                    <Menu.Item key="add-metadata" onClick={onAddMetadata}>Add Metadata</Menu.Item>}
                  <Menu.Item
                    danger
                    disabled={!isDeletable}
                    key="deregister-version"
                    onClick={() => showConfirmDelete(modelVersion)}>
                  Deregister Version
                  </Menu.Item>
                </Menu>
              )}
              trigger={[ 'click' ]}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </div>
        </div>
        <InfoBox rows={infoRows} separator={false} />
      </div>
    </header>
  );
};

export default ModelVersionHeader;
