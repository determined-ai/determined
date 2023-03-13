import { LeftOutlined } from '@ant-design/icons';
import { Dropdown, Modal, Space } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import InfoBox, { InfoRow } from 'components/InfoBox';
import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Avatar from 'components/kit/UserAvatar';
import Link from 'components/Link';
import TimeAgo from 'components/TimeAgo';
import useModalModelDownload from 'hooks/useModal/Model/useModalModelDownload';
import useModalModelEdit from 'hooks/useModal/Model/useModalModelEdit';
import useModalModelVersionDelete from 'hooks/useModal/Model/useModalModelVersionDelete';
import usePermissions from 'hooks/usePermissions';
import { WorkspaceDetailsTab } from 'pages/WorkspaceDetails';
import { paths } from 'routes/utils';
import CopyButton from 'shared/components/CopyButton';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner';
import { formatDatetime } from 'shared/utils/datetime';
import { copyToClipboard } from 'shared/utils/dom';
import usersStore from 'stores/users';
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
  onSaveDescription: (editedNotes: string) => Promise<void>;
  onSaveName: (editedName: string) => Promise<void>;
  onUpdateTags: (newTags: string[]) => Promise<void>;
  workspace: Workspace;
}

const ModelVersionHeader: React.FC<Props> = ({
  modelVersion,
  workspace,
  onSaveDescription,
  onUpdateTags,
  onSaveName,
}: Props) => {
  const loadableUsers = useObservable(usersStore.getUsers());
  const users = Loadable.map(loadableUsers, ({ users }) => users);
  const [showUseInNotebook, setShowUseInNotebook] = useState(false);

  const { contextHolder: modalModelDownloadContextHolder, modalOpen: openModelDownload } =
    useModalModelDownload();

  const { contextHolder: modalModelVersionDeleteContextHolder, modalOpen: openModalVersionDelete } =
    useModalModelVersionDelete();

  const { contextHolder: modalModelNameEditContextHolder, modalOpen: openModelNameEdit } =
    useModalModelEdit({ modelName: modelVersion.name ?? '', onSaveName });

  const handleDownloadModel = useCallback(() => {
    openModelDownload(modelVersion);
  }, [modelVersion, openModelDownload]);

  const { canDeleteModelVersion, canModifyModelVersion } = usePermissions();

  const infoRows: InfoRow[] = useMemo(() => {
    if (Loadable.isLoading(users)) return [];

    const user = users.data.find((user) => user.id === modelVersion.userId);
    return [
      {
        content: (
          <Space>
            <Avatar user={user} />
            {getDisplayName(user)}
            on {formatDatetime(modelVersion.creationTime, { format: 'MMM D, YYYY' })}
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
          <Input
            defaultValue={modelVersion.comment ?? ''}
            disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}
            placeholder={modelVersion.model.archived ? 'Archived' : 'Add description...'}
            onBlur={(e) => {
              const newValue = e.currentTarget.value;
              onSaveDescription(newValue);
            }}
            onPressEnter={(e) => {
              e.currentTarget.blur();
            }}
          />
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
  }, [modelVersion, onSaveDescription, onUpdateTags, users, canModifyModelVersion]);

  const handleDelete = useCallback(() => {
    openModalVersionDelete(modelVersion);
  }, [openModalVersionDelete, modelVersion]);

  const actions: Action[] = useMemo(() => {
    const items: Action[] = [
      {
        danger: false,
        disabled: false,
        key: 'download-model',
        onClick: handleDownloadModel,
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
        onClick: openModelNameEdit,
        text: 'Edit',
      },
    ];
    if (canDeleteModelVersion({ modelVersion })) {
      items.push({
        danger: true,
        disabled: false,
        key: 'deregister-version',
        onClick: handleDelete,
        text: 'Deregister Version',
      });
    }
    return items;
  }, [
    modelVersion,
    canModifyModelVersion,
    openModelNameEdit,
    handleDownloadModel,
    canDeleteModelVersion,
    handleDelete,
  ]);

  const referenceText = useMemo(() => {
    const escapedModelName = modelVersion.model.name.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
    return `from determined.experimental import Determined
client = Determined()
model_entry = client.get_model("${escapedModelName}")
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
my_model.load_state_dict(ckpt['models_state_dict'][0])`;
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
        <Spinner spinning>
          <InfoBox rows={infoRows} separator={false} />
        </Spinner>
      </div>
      {modalModelDownloadContextHolder}
      {modalModelVersionDeleteContextHolder}
      {modalModelNameEditContextHolder}
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
