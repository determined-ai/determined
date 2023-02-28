import { Typography } from 'antd';
import { FilterValue, SorterResult, TablePaginationConfig } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Input from 'components/kit/Input';
import TagList, { tagsActionHelper } from 'components/kit/Tags';
import MetadataCard from 'components/Metadata/MetadataCard';
import NotesCard from 'components/NotesCard';
import Page from 'components/Page';
import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  modelVersionNameRenderer,
  modelVersionNumberRenderer,
  relativeTimeRenderer,
  userRenderer,
} from 'components/Table/Table';
import useModalModelDownload from 'hooks/useModal/Model/useModalModelDownload';
import useModalModelVersionDelete from 'hooks/useModal/Model/useModalModelVersionDelete';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import {
  archiveModel,
  getModelDetails,
  patchModel,
  patchModelVersion,
  unarchiveModel,
} from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { isAborted, isNotFound, validateDetApiEnum } from 'shared/utils/service';
import { useUsers } from 'stores/users';
import { useEnsureWorkspacesFetched, useWorkspaces } from 'stores/workspaces';
import { Metadata, ModelVersion, ModelVersions } from 'types';
import handleError from 'utils/error';
import { Loadable, NotLoaded } from 'utils/loadable';

import css from './ModelDetails.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  isOfSortKey,
  Settings,
} from './ModelDetails/ModelDetails.settings';
import ModelHeader from './ModelDetails/ModelHeader';
import ModelVersionActionDropdown from './ModelDetails/ModelVersionActionDropdown';

type Params = {
  modelId: string;
};

const ModelDetails: React.FC = () => {
  const canceler = useRef(new AbortController());
  const [model, setModel] = useState<ModelVersions>();
  const modelId = decodeURIComponent(useParams<Params>().modelId ?? '');
  const [isLoading, setIsLoading] = useState(true);
  const [pageError, setPageError] = useState<Error>();
  const [total, setTotal] = useState(0);
  const pageRef = useRef<HTMLElement>(null);
  const users = Loadable.match(useUsers(), {
    Loaded: (usersPagination) => usersPagination.users,
    NotLoaded: () => [],
  });
  const ensureWorkspacesFetched = useEnsureWorkspacesFetched(canceler.current);
  const lodableWorkspaces = useWorkspaces();
  const workspace = Loadable.getOrElse([], lodableWorkspaces).find(
    (ws) => ws.id === model?.model.workspaceId,
  );

  const { canModifyModel, canModifyModelVersion } = usePermissions();

  const {
    settings,
    isLoading: isLoadingSettings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig(modelId));

  const fetchModel = useCallback(async () => {
    if (!settings) return;

    try {
      const modelData = await getModelDetails({
        limit: settings.tableLimit,
        modelName: modelId,
        offset: settings.tableOffset,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        sortBy: validateDetApiEnum(V1GetModelVersionsRequestSortBy, settings.sortKey),
      });
      setTotal(modelData?.pagination.total || 0);
      setModel((prev) => (!isEqual(modelData, prev) ? modelData : prev));
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    } finally {
      setIsLoading(false);
    }
  }, [modelId, pageError, settings]);

  const { contextHolder: modalModelDownloadContextHolder, modalOpen: openModelDownload } =
    useModalModelDownload();

  const { contextHolder: modalModelVersionDeleteContextHolder, modalOpen: openModelVersionDelete } =
    useModalModelVersionDelete();

  usePolling(fetchModel, { rerunOnNewFn: true });

  useEffect(() => {
    setIsLoading(true);
    fetchModel();
    ensureWorkspacesFetched();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const downloadModel = useCallback(
    (version: ModelVersion) => {
      openModelDownload(version);
    },
    [openModelDownload],
  );

  const deleteModelVersion = useCallback(
    (version: ModelVersion) => {
      openModelVersionDelete(version);
    },
    [openModelVersionDelete],
  );

  const saveModelVersionTags = useCallback(
    async (modelName: string, versionNum: number, tags: string[]) => {
      try {
        await patchModelVersion({
          body: { labels: tags, modelName },
          modelName,
          versionNum: versionNum,
        });
        await fetchModel();
      } catch (e) {
        handleError(e, {
          publicSubject: `Unable to update model version ${versionNum} tags.`,
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModel],
  );

  const saveVersionDescription = useCallback(
    async (editedDescription: string, versionNum: number) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModelVersion({
            body: { comment: editedDescription, modelName },
            modelName,
            versionNum: versionNum,
          });
        }
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save version description.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [model?.model.name],
  );

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ModelVersion) => (
      <div className={css.tagsRenderer}>
        <Typography.Text
          ellipsis={{
            tooltip: <TagList disabled tags={record.labels ?? []} />,
          }}>
          <div>
            <TagList
              compact
              disabled={record.model.archived}
              tags={record.labels ?? []}
              onAction={tagsActionHelper(record.labels ?? [], (tags) =>
                saveModelVersionTags(record.model.name, record.version, tags),
              )}
            />
          </div>
        </Typography.Text>
      </div>
    );

    const actionRenderer = (_: string, record: ModelVersion) => (
      <ModelVersionActionDropdown
        version={record}
        onDelete={() => deleteModelVersion(record)}
        onDownload={() => downloadModel(record)}
      />
    );

    const descriptionRenderer = (value: string, record: ModelVersion) => (
      <Input
        className={css.descriptionRenderer}
        defaultValue={record.comment ?? ''}
        disabled={record.model.archived || !canModifyModelVersion({ modelVersion: record })}
        placeholder={record.model.archived ? 'Archived' : 'Add description...'}
        title={record.model.archived ? 'Archived description' : 'Edit description'}
        onBlur={(e) => {
          const newDesc = e.currentTarget.value;
          saveVersionDescription(newDesc, record.version);
        }}
        onPressEnter={(e) => {
          // when enter is pressed,
          // input box gets blurred and then value will be saved in onBlur
          e.currentTarget.blur();
        }}
      />
    );

    return [
      {
        dataIndex: 'version',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['version'],
        key: V1GetModelVersionsRequestSortBy.VERSION,
        render: modelVersionNumberRenderer,
        sorter: true,
        title: 'V',
      },
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        render: modelVersionNameRenderer,
        title: 'Name',
      },
      {
        dataIndex: 'description',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['description'],
        render: descriptionRenderer,
        title: 'Description',
      },
      {
        dataIndex: 'lastUpdatedTime',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['lastUpdatedTime'],
        render: (date: Date, record: ModelVersion) =>
          relativeTimeRenderer(date ?? record.creationTime),
        title: 'Last updated',
      },
      {
        dataIndex: 'user',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
        key: 'user',
        render: (_, r) => userRenderer(users.find((u) => u.id === r.userId)),
        title: 'User',
      },
      {
        dataIndex: 'tags',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['tags'],
        render: tagsRenderer,
        title: 'Tags',
      },
      {
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        render: actionRenderer,
        title: '',
      },
    ] as ColumnDef<ModelVersion>[];
  }, [
    deleteModelVersion,
    downloadModel,
    saveModelVersionTags,
    saveVersionDescription,
    users,
    canModifyModelVersion,
  ]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<ModelVersion> | SorterResult<ModelVersion>[],
    ) => {
      if (Array.isArray(tableSorter) || !settings.tableOffset) return;

      const { columnKey, order } = tableSorter as SorterResult<ModelVersion>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: isOfSortKey(columnKey) ? columnKey : V1GetModelVersionsRequestSortBy.UNSPECIFIED,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      const shouldPush = settings.tableOffset !== newSettings.tableOffset;
      updateSettings(newSettings, shouldPush);
    },
    [columns, settings.tableOffset, updateSettings],
  );

  const saveMetadata = useCallback(
    async (editedMetadata: Metadata) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModel({
            body: { metadata: editedMetadata, name: modelName },
            modelName,
          });
        }
        await fetchModel();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save metadata.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModel, model?.model.name],
  );

  const saveDescription = useCallback(
    async (editedDescription: string) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModel({
            body: { description: editedDescription, name: modelName },
            modelName,
          });
        }
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save description.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [model?.model.name],
  );

  const saveName = useCallback(
    async (editedName: string) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModel({
            body: { name: editedName },
            modelName,
          });
        }
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save name.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [model?.model.name],
  );

  const saveNotes = useCallback(
    async (editedNotes: string) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModel({
            body: { name: modelName, notes: editedNotes },
            modelName,
          });
        }
        await fetchModel();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to update notes.',
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [model?.model.name, fetchModel],
  );

  const saveModelTags = useCallback(
    async (editedTags: string[]) => {
      try {
        const modelName = model?.model.name;
        if (modelName) {
          await patchModel({
            body: { labels: editedTags, name: modelName },
            modelName,
          });
          fetchModel();
        }
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to update model tags.',
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModel, model?.model.name],
  );

  const switchArchive = useCallback(() => {
    const modelName = model?.model.name;
    if (modelName) {
      if (model?.model.archived) {
        unarchiveModel({ modelName });
      } else {
        archiveModel({ modelName });
      }
    }
  }, [model?.model.archived, model?.model.name]);

  const actionDropdown = useCallback(
    ({
      record,
      onVisibleChange,
      children,
    }: {
      children: React.ReactNode;
      onVisibleChange?: (visible: boolean) => void;
      record: ModelVersion;
    }) => (
      <ModelVersionActionDropdown
        trigger={['contextMenu']}
        version={record}
        onDelete={() => deleteModelVersion(record)}
        onDownload={() => downloadModel(record)}
        onVisibleChange={onVisibleChange}>
        {children}
      </ModelVersionActionDropdown>
    ),
    [deleteModelVersion, downloadModel],
  );

  if (!modelId) {
    return <Message title="Model name is empty" />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `Unable to fetch model ${modelId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!model || lodableWorkspaces === NotLoaded) {
    return <Spinner tip={`Loading model ${modelId} details...`} />;
  }

  return (
    <Page
      containerRef={pageRef}
      docTitle="Model Details"
      headerComponent={
        <ModelHeader
          model={model.model}
          workspace={workspace}
          onSaveDescription={saveDescription}
          onSaveName={saveName}
          onSwitchArchive={switchArchive}
          onUpdateTags={saveModelTags}
        />
      }
      id="modelDetails"
      notFound={pageError && isNotFound(pageError)}>
      <div className={css.base}>
        {model.modelVersions.length === 0 ? (
          <div className={css.noVersions}>
            <p className={css.header}>No Model Versions</p>
            <p className={css.subtext}>
              Register a checkpoint from an experiment to add it to this model
            </p>
          </div>
        ) : (
          <InteractiveTable
            columns={columns}
            containerRef={pageRef}
            ContextMenu={actionDropdown}
            dataSource={model.modelVersions}
            loading={isLoading || isLoadingSettings}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="version"
            settings={settings as InteractiveTableSettings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings as UpdateSettings}
            onChange={handleTableChange}
          />
        )}
        <NotesCard
          disabled={model.model.archived || !canModifyModel({ model: model.model })}
          notes={model.model.notes ?? ''}
          onSave={saveNotes}
        />
        <MetadataCard
          disabled={model.model.archived || !canModifyModel({ model: model.model })}
          metadata={model.model.metadata}
          onSave={saveMetadata}
        />
      </div>
      {modalModelDownloadContextHolder}
      {modalModelVersionDeleteContextHolder}
    </Page>
  );
};

export default ModelDetails;
