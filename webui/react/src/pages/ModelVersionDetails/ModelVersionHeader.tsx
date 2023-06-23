import { Modal, Space, Typography } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import Button from 'components/kit/Button';
import ClipboardButton from 'components/kit/ClipboardButton';
import Dropdown, { MenuOption } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Avatar from 'components/kit/UserAvatar';
import ModelDownloadModal from 'components/ModelDownloadModal';
import ModelVersionDeleteModal from 'components/ModelVersionDeleteModal';
import ModelVersionEditModal from 'components/ModelVersionEditModal';
import Spinner from 'components/Spinner';
import TimeAgo from 'components/TimeAgo';
import usePermissions from 'hooks/usePermissions';
import userStore from 'stores/users';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/datetime';
import { Loadable } from 'utils/loadable';
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
          <ClipboardButton getContent={() => referenceText} />
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
