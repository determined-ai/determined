import { Modal, Space, Typography } from 'antd';
import Button from 'hew/Button';
import CodeSample from 'hew/CodeSample';
import Dropdown, { MenuOption } from 'hew/Dropdown';
import Glossary, { InfoRow } from 'hew/Glossary';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Nameplate from 'hew/Nameplate';
import Spinner from 'hew/Spinner';
import Tags, { tagsActionHelper } from 'hew/Tags';
import { useTheme } from 'hew/Theme';
import React, { useCallback, useMemo, useState } from 'react';

import ModelDownloadModal from 'components/ModelDownloadModal';
import ModelVersionDeleteModal from 'components/ModelVersionDeleteModal';
import ModelVersionEditModal from 'components/ModelVersionEditModal';
import TimeAgo from 'components/TimeAgo';
import Avatar from 'components/UserAvatar';
import usePermissions from 'hooks/usePermissions';
import userStore from 'stores/users';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/datetime';
import { useObservable } from 'utils/observable';
import { getDisplayName } from 'utils/user';

import css from './ModelVersionHeader.module.scss';

interface Props {
  modelVersion: ModelVersion;
  fetchModelVersion: () => Promise<void>;
  onUpdateTags: (newTags: string[]) => Promise<void>;
}

const MenuKey = {
  DeregisterVersion: 'Deregister Version',
  DownloadModel: 'Download',
  EditModelVersionName: 'Edit',
  UseInNotebook: 'Use in Notebook',
} as const;

const ModelVersionHeader: React.FC<Props> = ({
  modelVersion,
  onUpdateTags,
  fetchModelVersion,
}: Props) => {
  const loadableUsers = useObservable(userStore.getUsers());
  const [showUseInNotebook, setShowUseInNotebook] = useState(false);

  const modelDownloadModal = useModal(ModelDownloadModal);
  const modelVersionDeleteModal = useModal(ModelVersionDeleteModal);
  const modelVersionEditModal = useModal(ModelVersionEditModal);

  const { canDeleteModelVersion, canModifyModelVersion } = usePermissions();

  const {
    themeSettings: { className: themeClass },
  } = useTheme();

  const infoRows: InfoRow[] = useMemo(
    () => [
      {
        label: 'Created by',
        value: (
          <Spinner data={loadableUsers}>
            {(users) => {
              const user = users.find((user) => user.id === modelVersion.userId);
              return (
                <Space>
                  <Nameplate
                    alias={getDisplayName(user)}
                    compact
                    icon={<Avatar user={user} />}
                    name={user?.username ?? 'Unavailable'}
                  />{' '}
                  on {formatDatetime(modelVersion.creationTime, { format: 'MMM D, YYYY' })}
                </Space>
              );
            }}
          </Spinner>
        ),
      },
      {
        label: 'Updated',
        value: (
          <TimeAgo datetime={new Date(modelVersion.lastUpdatedTime ?? modelVersion.creationTime)} />
        ),
      },
      {
        label: 'Description',
        value: (
          <div>
            {(modelVersion.comment ?? '') || (
              <Typography.Text
                disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}>
                N/A
              </Typography.Text>
            )}
          </div>
        ),
      },
      {
        label: 'Tags',
        value: (
          <Tags
            disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}
            ghost={false}
            tags={modelVersion.labels ?? []}
            onAction={tagsActionHelper(modelVersion.labels ?? [], onUpdateTags)}
          />
        ),
      },
    ],
    [loadableUsers, modelVersion, onUpdateTags, canModifyModelVersion],
  );

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

  const menu = useMemo(() => {
    const items: MenuOption[] = [
      {
        key: MenuKey.DownloadModel,
        label: MenuKey.DownloadModel,
      },
      {
        key: MenuKey.UseInNotebook,
        label: MenuKey.UseInNotebook,
      },
      {
        disabled: modelVersion.model.archived || !canModifyModelVersion({ modelVersion }),
        key: MenuKey.EditModelVersionName,
        label: MenuKey.EditModelVersionName,
      },
    ];
    if (canDeleteModelVersion({ modelVersion })) {
      items.push({
        danger: true,
        key: MenuKey.DeregisterVersion,
        label: MenuKey.DeregisterVersion,
      });
    }
    return items;
  }, [canDeleteModelVersion, canModifyModelVersion, modelVersion]);

  const handleDropdown = useCallback(
    (key: string | number) => {
      switch (key) {
        case MenuKey.DeregisterVersion:
          modelVersionDeleteModal.open();
          break;
        case MenuKey.DownloadModel:
          modelDownloadModal.open();
          break;
        case MenuKey.EditModelVersionName:
          modelVersionEditModal.open();
          break;
        case MenuKey.UseInNotebook:
          setShowUseInNotebook(true);
          break;
        default:
          return;
      }
    },
    [modelDownloadModal, modelVersionEditModal, modelVersionDeleteModal],
  );

  return (
    <header className={css.base}>
      <div className={css.headerContent}>
        <div className={css.mainRow}>
          <div className={css.title}>
            <div className={css.versionBox}>V{modelVersion.version}</div>
            <h1 className={css.versionName}>
              {modelVersion.name ? modelVersion.name : `Version ${modelVersion.version}`}
            </h1>
          </div>
          <div className={css.buttons}>
            {menu.slice(0, 2).map((item) => (
              <Button
                danger={item.danger}
                disabled={item.disabled}
                key={item.key}
                onClick={() => handleDropdown(item.key)}>
                {item.label}
              </Button>
            ))}
            <Dropdown menu={menu.slice(2)} onClick={handleDropdown}>
              <Button
                icon={<Icon name="overflow-horizontal" size="small" title="Action menu" />}
                type="text"
              />
            </Dropdown>
          </div>
        </div>
        <Glossary content={infoRows} />
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
        wrapClassName={themeClass}
        onCancel={() => setShowUseInNotebook(false)}>
        <div className={css.topLine}>
          <p>Reference this model in a notebook</p>
        </div>
        <CodeSample text={referenceText} />
        <p>Copy/paste code into a notebook cell</p>
      </Modal>
    </header>
  );
};

export default ModelVersionHeader;
