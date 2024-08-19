import Button from 'hew/Button';
import Dropdown, { MenuOption } from 'hew/Dropdown';
import Glossary, { InfoRow } from 'hew/Glossary';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Nameplate from 'hew/Nameplate';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import Tags, { tagsActionHelper } from 'hew/Tags';
import { Title } from 'hew/Typography';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import ModelDownloadModal from 'components/ModelDownloadModal';
import ModelVersionDeleteModal from 'components/ModelVersionDeleteModal';
import ModelVersionEditModal from 'components/ModelVersionEditModal';
import TimeAgo from 'components/TimeAgo';
import UseNotebookModalComponent from 'components/UseNotebookModalComponent';
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
  const useNotebookModal = useModal(UseNotebookModalComponent);

  const { canDeleteModelVersion, canModifyModelVersion } = usePermissions();

  useEffect(() => {
    if (showUseInNotebook) useNotebookModal.open();
  }, [showUseInNotebook, useNotebookModal]);

  const infoRows: InfoRow[] = useMemo(
    () => [
      {
        label: 'Created by',
        value: (
          <Spinner data={loadableUsers}>
            {(users) => {
              const user = users.find((user) => user.id === modelVersion.userId);
              return (
                <Row>
                  <Nameplate
                    alias={getDisplayName(user)}
                    compact
                    icon={<Avatar user={user} />}
                    name={user?.username ?? 'Unavailable'}
                  />{' '}
                  on {formatDatetime(modelVersion.creationTime, { format: 'MMM D, YYYY' })}
                </Row>
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
        value: <div>{modelVersion.comment || <span>N/A</span>}</div>,
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
        <Row justifyContent="space-between" wrap>
          <div className={css.mainRow}>
            <div className={css.versionBox}>V{modelVersion.version}</div>
            <Title truncate={{ tooltip: true }}>
              {modelVersion.name || `Version ${modelVersion.version}`}
            </Title>
          </div>
          <Row wrap>
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
          </Row>
        </Row>
        <Glossary content={infoRows} />
      </div>
      <modelDownloadModal.Component modelVersion={modelVersion} />
      <modelVersionDeleteModal.Component modelVersion={modelVersion} />
      <modelVersionEditModal.Component
        fetchModelVersion={fetchModelVersion}
        modelVersion={modelVersion}
      />
      <useNotebookModal.Component
        modelVersion={modelVersion}
        onClose={() => setShowUseInNotebook(false)}
      />
    </header>
  );
};

export default ModelVersionHeader;
