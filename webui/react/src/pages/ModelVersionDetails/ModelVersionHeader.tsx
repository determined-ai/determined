import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Dropdown, Menu, Modal, Space } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Avatar from 'components/Avatar';
import CopyButton from 'components/CopyButton';
import DownloadModelModal from 'components/DownloadModelModal';
import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import Link from 'components/Link';
import { relativeTimeRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/datetime';
import { copyToClipboard } from 'utils/dom';

import css from './ModelVersionHeader.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onDeregisterVersion: () => void;
  onSaveDescription: (editedNotes: string) => Promise<void>;
  onSaveName: (editedName: string) => Promise<void>;
  onUpdateTags: (newTags: string[]) => Promise<void>;
}

const ModelVersionHeader: React.FC<Props> = (
  {
    modelVersion, onDeregisterVersion,
    onSaveDescription, onUpdateTags, onSaveName,
  }: Props,
) => {
  const { auth: { user } } = useStore();
  const [ showUseInNotebook, setShowUseInNotebook ] = useState(false);
  const [ showDownloadModel, setShowDownloadModel ] = useState(false);

  const isDeletable = user?.isAdmin
        || user?.username === modelVersion.model.username
        || user?.username === modelVersion.username;

  const showConfirmDelete = useCallback(() => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this version "Version ${modelVersion.version}" 
            from this model?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Version',
      okType: 'danger',
      onOk: onDeregisterVersion,
      title: 'Confirm Delete',
    });
  }, [ onDeregisterVersion, modelVersion.version ]);

  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content: (
        <Space>
          {modelVersion.username ? (
            <Avatar name={modelVersion.username} />
          ) : (
            <Avatar name={modelVersion.model.username} />
          )}
          {modelVersion.username ? modelVersion.username : modelVersion.model.username}
          on {formatDatetime(modelVersion.creationTime, { format: 'MMM D, YYYY' })}
        </Space>
      ),
      label: 'Created by',
    },
    {
      content: relativeTimeRenderer(
        new Date(modelVersion.lastUpdatedTime ?? modelVersion.creationTime),
      ),
      label: 'Updated',
    },
    {
      content: (
        <InlineEditor
          disabled={modelVersion.model.archived}
          placeholder="Add description..."
          value={modelVersion.comment ?? ''}
          onSave={onSaveDescription}
        />
      ),
      label: 'Description',
    },
    {
      content: (
        <TagList
          disabled={modelVersion.model.archived}
          ghost={false}
          tags={modelVersion.labels ?? []}
          onChange={onUpdateTags}
        />
      ),
      label: 'Tags',
    } ] as InfoRow[];
  }, [ modelVersion, onSaveDescription, onUpdateTags ]);

  const actions = useMemo(() => {
    return [
      {
        danger: false,
        disabled: false,
        key: 'download-model',
        onClick: () => setShowDownloadModel(true),
        text: 'Download',
      },
      {
        danger: false,
        disabled: false,
        key: 'use-in-notebook',
        onClick: () => setShowUseInNotebook(true),
        text: 'Use in Notebook',
      },
      {
        danger: true,
        disabled: !isDeletable,
        key: 'deregister-version',
        onClick: showConfirmDelete,
        text: 'Deregister Version',
      },
    ];
  }, [ isDeletable, showConfirmDelete ]);

  const referenceText = useMemo(() => {
    return (
      `from determined.experimental import Determined
client = Determined()
model_entry = client.get_model_by_name("${modelVersion.model.name}")
version = model_entry.get_version(${modelVersion.version})
ckpt = version.checkpoint
      
################ Approach 1 ################
# You can load the trial directly without having to instantiate the model. 
# The trial should have the model as an attribute.
trial = ckpt.load() 
      
################ Approach 2 ################
# You can download the checkpoint and load the model state manually.
ckpt_path = ckpt.download()
ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))
# assuming your model is already instantiated, you can then load the state_dict
my_model.load_state_dict(ckpt['models_state_dict'][0])`);
  }, [ modelVersion ]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(referenceText);
  }, [ referenceText ]);

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
            <h1 className={css.versionName}>
              <InlineEditor
                allowClear={false}
                disabled={modelVersion.model.archived}
                placeholder="Add name..."
                value={modelVersion.name ? modelVersion.name : `Version ${modelVersion.version}`}
                onSave={onSaveName}
              />
            </h1>
          </div>
          <div className={css.buttons}>
            {actions.slice(0, 2).map(action => (
              <Button
                className={css.buttonAction}
                danger={action.danger}
                disabled={action.disabled}
                key={action.key}
                onClick={action.onClick}>
                {action.text}
              </Button>
            ))}
            <Dropdown
              overlay={(
                <Menu className={css.overflow}>
                  {actions.map(action => (
                    <Menu.Item
                      className={css.overflowAction}
                      danger={action.danger}
                      disabled={action.disabled}
                      key={action.key}
                      onClick={action.onClick}>
                      {action.text}
                    </Menu.Item>
                  ))}
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
      <DownloadModelModal
        modelVersion={modelVersion}
        visible={showDownloadModel}
        onClose={() => setShowDownloadModel(false)}
      />
      <Modal
        className={css.useNotebookModal}
        footer={null}
        title="Use in Notebook"
        visible={showUseInNotebook}
        onCancel={() => setShowUseInNotebook(false)}>
        <div className={css.topLine}>
          <p>Reference this model in a notebook</p>
          <CopyButton onCopy={handleCopy} />
        </div>
        <pre className={css.codeSample}><code>{referenceText}</code></pre>
        <p>Copy/paste code into a notebook cell</p>
      </Modal>
    </header>
  );
};

export default ModelVersionHeader;
