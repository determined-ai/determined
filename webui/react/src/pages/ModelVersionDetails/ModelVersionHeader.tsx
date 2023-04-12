import { LeftOutlined } from '@ant-design/icons';
import { Dropdown, Modal, Space, Typography } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Avatar from 'components/kit/UserAvatar';
import Link from 'components/Link';
import ModelDownloadModal from 'components/ModelDownloadModal';
import ModelVersionDeleteModal from 'components/ModelVersionDeleteModal';
import ModelVersionEditModal from 'components/ModelVersionEditModal';
import TimeAgo from 'components/TimeAgo';
import usePermissions from 'hooks/usePermissions';
import { WorkspaceDetailsTab } from 'pages/WorkspaceDetails';
import { paths } from 'routes/utils';
import CopyButton from 'shared/components/CopyButton';
import Spinner from 'shared/components/Spinner';
import { formatDatetime } from 'shared/utils/datetime';
import { copyToClipboard } from 'shared/utils/dom';
import userStore from 'stores/users';
import { ModelVersion, Workspace } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { getDisplayName } from 'utils/user';

import css from './ModelVersionHeader.module.scss';

type Action = {
  danger: boolean;
  disabled: boolean;
  key: string;
  onClick: () => void;
  text: string;
};

interface Props {
  modelVersion: ModelVersion;
  fetchModelVersion: () => Promise<void>;
  onUpdateTags: (newTags: string[]) => Promise<void>;
  workspace: Workspace;
}

const ModelVersionHeader: React.FC<Props> = ({
  modelVersion,
  workspace,
  onUpdateTags,
  fetchModelVersion,
}: Props) => {
  const loadableUsers = useObservable(userStore.getUsers());
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const [showUseInNotebook, setShowUseInNotebook] = useState(false);

  const modelDownloadModal = useModal(ModelDownloadModal);
  const modelVersionDeleteModal = useModal(ModelVersionDeleteModal);
  const modelVersionEditModal = useModal(ModelVersionEditModal);

  const { canDeleteModelVersion, canModifyModelVersion } = usePermissions();

  const infoRows: InfoRow[] = useMemo(() => {
    const user = users.find((user) => user.id === modelVersion.userId);

    return [
      {
        content: (
          <Space>
            <Spinner conditionalRender spinning={Loadable.isLoading(loadableUsers)}>
              <>
                <Avatar user={user} />
                {getDisplayName(user)} on{' '}
                {formatDatetime(modelVersion.creationTime, { format: 'MMM D, YYYY' })}
              </>
            </Spinner>
          </Space>
        ),
        label: 'Created by',
      },
      {
        content: (
          <TimeAgo datetime={new Date(modelVersion.lastUpdatedTime ?? modelVersion.creationTime)} />
        ),
        label: 'Updated',
      },
      {
        content: (
          <div>
            {(modelVersion.comment ?? '') || (
              <Typography.Text
                disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}>
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
            disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}
            ghost={false}
            tags={modelVersion.labels ?? []}
            onAction={tagsActionHelper(modelVersion.labels ?? [], onUpdateTags)}
          />
        ),
        label: 'Tags',
      },
    ] as InfoRow[];
  }, [loadableUsers, modelVersion, onUpdateTags, users, canModifyModelVersion]);

  const actions: Action[] = useMemo(() => {
    const items: Action[] = [
      {
        danger: false,
        disabled: false,
        key: 'download-model',
        onClick: () => modelDownloadModal.open(),
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
        danger: false,
        disabled: modelVersion.model.archived || !canModifyModelVersion({ modelVersion }),
        key: 'edit-model-version-name',
        onClick: () => modelVersionEditModal.open(),
        text: 'Edit',
      },
    ];
    if (canDeleteModelVersion({ modelVersion })) {
      items.push({
        danger: true,
        disabled: false,
        key: 'deregister-version',
        onClick: () => modelVersionDeleteModal.open(),
        text: 'Deregister Version',
      });
    }
    return items;
  }, [
    modelVersion,
    canModifyModelVersion,
    canDeleteModelVersion,
    modelDownloadModal,
    modelVersionEditModal,
    modelVersionDeleteModal,
  ]);

  const referenceText = useMemo(() => {
    const escapedModelName = modelVersion.model.name.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
    return `import determined as det
from determined.experimental import client

model_entry = client.get_model("${escapedModelName}")
version = model_entry.get_version(${modelVersion.version})
ckpt = version.checkpoint
path = ckpt.download()

# Load a PyTorchTrial from a checkpoint:
from determined import pytorch
my_trial = \\
    pytorch.load_trial_from_checkpoint_path(path)

# Load a Keras model from TFKerasTrial checkpoint:
from determined import keras
model = keras.load_model_from_checkpoint_path(path)

# Import your checkpointed code:
with det.import_from_path(path + "/code"):
    import my_model_def as ckpt_model_def
`;
  }, [modelVersion]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(referenceText);
  }, [referenceText]);

  const menu: DropDownProps['menu'] = useMemo(() => {
    const onItemClick: MenuProps['onClick'] = (e) => {
      const action = actions.find((ac) => ac.key === e.key) as Action;
      action.onClick();
    };

    const menuItems: MenuProps['items'] = actions.map((action) => ({
      className: css.overflowAction,
      danger: action.danger,
      disabled: action.disabled,
      key: action.key,
      label: action.text,
    }));

    return { className: css.overflow, items: menuItems, onClick: onItemClick };
  }, [actions]);

  return (
    <header className={css.base}>
      <div className={css.breadcrumbs}>
        <Breadcrumb separator="">
          <Breadcrumb.Item>
            <Link path={paths.modelDetails(String(modelVersion.model.id))}>
              <LeftOutlined style={{ marginRight: 10 }} />
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link
              path={
                workspace.id === 1 ? paths.projectDetails(1) : paths.workspaceDetails(workspace.id)
              }>
              {workspace.name}
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            <Link path={paths.workspaceDetails(workspace.id, WorkspaceDetailsTab.ModelRegistry)}>
              Model Registry
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>
            <Link path={paths.modelDetails(String(modelVersion.model.id))}>
              {modelVersion.model.name} ({modelVersion.model.id})
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>Version {modelVersion.version}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div className={css.headerContent}>
        <div className={css.mainRow}>
          <div className={css.title}>
            <div className={css.versionBox}>V{modelVersion.version}</div>
            <h1 className={css.versionName}>
              {modelVersion.name ? modelVersion.name : `Version ${modelVersion.version}`}
            </h1>
          </div>
          <div className={css.buttons}>
            {actions.slice(0, 2).map((action) => (
              <Button
                danger={action.danger}
                disabled={action.disabled}
                key={action.key}
                onClick={action.onClick}>
                {action.text}
              </Button>
            ))}
            <Dropdown menu={menu} trigger={['click']}>
              <Button type="text">
                <Icon name="overflow-horizontal" size="tiny" />
              </Button>
            </Dropdown>
          </div>
        </div>
        <InfoBox rows={infoRows} separator={false} />
      </div>
      <modelDownloadModal.Component modelVersion={modelVersion} />
      <modelVersionDeleteModal.Component modelVersion={modelVersion} />
      <modelVersionEditModal.Component
        fetchModelVersion={fetchModelVersion}
        modelVersion={modelVersion}
      />
      <Modal
        className={css.useNotebookModal}
        footer={null}
        open={showUseInNotebook}
        title="Use in Notebook"
        onCancel={() => setShowUseInNotebook(false)}>
        <div className={css.topLine}>
          <p>Reference this model in a notebook</p>
          <CopyButton onCopy={handleCopy} />
        </div>
        <pre className={css.codeSample}>
          <code>{referenceText}</code>
        </pre>
        <p>Copy/paste code into a notebook cell</p>
      </Modal>
    </header>
  );
};

export default ModelVersionHeader;
