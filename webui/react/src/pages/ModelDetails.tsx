import { Typography } from 'antd';
import { FilterValue, SorterResult, TablePaginationConfig } from 'antd/lib/table/interface';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Input from 'components/kit/Input';
import Notes from 'components/kit/Notes';
import Spinner from 'components/kit/Spinner';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import Message, { MessageType } from 'components/kit/Message';
import MetadataCard from 'components/Metadata/MetadataCard';
import Page, { BreadCrumbRoute } from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  modelVersionNameRenderer,
  modelVersionNumberRenderer,
  relativeTimeRenderer,
  userRenderer,
} from 'components/Table/Table';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import {
  archiveModel,
  getModelDetails,
  patchModel,
  patchModelVersion,
  unarchiveModel,
} from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { Metadata, ModelItem, ModelVersion, Note } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { isAborted, isNotFound, validateDetApiEnum } from 'utils/service';

import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  isOfSortKey,
  Settings,
} from './ModelDetails/ModelDetails.settings';
import ModelHeader from './ModelDetails/ModelHeader';
import ModelVersionActionDropdown from './ModelDetails/ModelVersionActionDropdown';
import css from './ModelDetails.module.scss';
import { WorkspaceDetailsTab } from './WorkspaceDetails';

type Params = {
  modelId: string;
};

const ModelDetails: React.FC = () => {
  const [model, setModel] = useState<ModelItem>();
  const [modelVersions, setModelVersions] = useState<ModelVersion[]>();
  const modelId = decodeURIComponent(useParams<Params>().modelId ?? '');
  const [isLoading, setIsLoading] = useState(true);
  const [pageError, setPageError] = useState<Error>();
  const [total, setTotal] = useState(0);
  const pageRef = useRef<HTMLElement>(null);
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const workspaces = useObservable(workspaceStore.workspaces);
  const workspace = Loadable.getOrElse(
    undefined,
    useObservable(workspaceStore.getWorkspace(model ? Loaded(model.workspaceId) : NotLoaded)),
  );

  const { canModifyModel, canModifyModelVersion, loading: rbacLoading } = usePermissions();
  const [permissionsByModelVersion, setPermissionsByModelVersion] = useState<
    Record<number, { canModify: boolean }>
  >({});

  useEffect(() => {
    const allPerm: Record<number, { canModify: boolean }> = {};
    modelVersions?.forEach((modelVersion) => {
      allPerm[modelVersion.id] = {
        canModify: canModifyModelVersion({ modelVersion }),
      };
    });
    setPermissionsByModelVersion((prev) => (_.isEqual(prev, allPerm) ? prev : allPerm));
  }, [canModifyModelVersion, modelVersions]);

  const config = useMemo(() => {
    return settingsConfig(modelId);
  }, [modelId]);

  const { settings, isLoading: isLoadingSettings, updateSettings } = useSettings<Settings>(config);

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
      setModel((prev) => (!_.isEqual(modelData?.model, prev) ? modelData?.model : prev));
      setModelVersions((prev) =>
        !_.isEqual(modelData?.modelVersions, prev) ? modelData?.modelVersions : prev,
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    } finally {
      setIsLoading(false);
    }
  }, [modelId, pageError, settings]);

  usePolling(fetchModel, { rerunOnNewFn: true });

  useEffect(() => {
    setIsLoading(true);
    fetchModel();
    return workspaceStore.fetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
        const modelName = model?.name;
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
    [model?.name],
  );

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ModelVersion) => (
      <div className={css.tagsRenderer}>
        <Typography.Text
          ellipsis={{
            tooltip: <Tags disabled tags={record.labels ?? []} />,
          }}>
          <div>
            <Tags
              compact
              disabled={record.model.archived || !permissionsByModelVersion[record.id]?.canModify}
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
      <ModelVersionActionDropdown version={record} />
    );

    const descriptionRenderer = (value: string, record: ModelVersion) => (
      <Input
        className={css.descriptionRenderer}
        defaultValue={record.comment ?? ''}
        disabled={record.model.archived || !permissionsByModelVersion[record.id]?.canModify}
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
  }, [permissionsByModelVersion, saveModelVersionTags, saveVersionDescription, users]);

  const tableIsLoading = useMemo(
    () => isLoading || isLoadingSettings,
    [isLoading, isLoadingSettings],
  );

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
      updateSettings(newSettings);
    },
    [columns, settings.tableOffset, updateSettings],
  );

  const saveMetadata = useCallback(
    async (editedMetadata: Metadata) => {
      try {
        const modelName = model?.name;
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
    [fetchModel, model?.name],
  );

  const saveNotes = useCallback(
    async (notes: Note) => {
      const editedNotes = notes.contents;
      try {
        const modelName = model?.name;
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
    [model?.name, fetchModel],
  );

  const saveModelTags = useCallback(
    async (editedTags: string[]) => {
      try {
        const modelName = model?.name;
        if (modelName) {
          await patchModel({
            body: { labels: editedTags, name: modelName },
            modelName,
          });
          await fetchModel();
        }
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to update model tags.',
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModel, model?.name],
  );

  const switchArchive = useCallback(() => {
    const modelName = model?.name;
    if (modelName) {
      if (model?.archived) {
        unarchiveModel({ modelName });
      } else {
        archiveModel({ modelName });
      }
    }
  }, [model?.archived, model?.name]);

  const actionDropdown = useCallback(
    ({ record, children }: { children: React.ReactNode; record: ModelVersion }) => (
      <ModelVersionActionDropdown isContextMenu version={record}>
        {children}
      </ModelVersionActionDropdown>
    ),
    [],
  );

  if (!modelId) {
    return <Message title="Model name is empty" />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `Unable to fetch model ${modelId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (pageError && isNotFound(pageError)) {
    return <PageNotFound />;
  } else if (!model || Loadable.isNotLoaded(workspaces) || rbacLoading) {
    return <Spinner spinning tip={`Loading model ${modelId} details...`} />;
  }

  const pageBreadcrumb: BreadCrumbRoute[] = [];
  if (workspace) {
    pageBreadcrumb.push(
      workspace.id === 1
        ? {
            breadcrumbName: 'Uncategorized Experiments',
            path: paths.projectDetails(1),
          }
        : {
            breadcrumbName: workspace.name,
            path: paths.workspaceDetails(workspace.id),
          },
      {
        breadcrumbName: 'Model Registry',
        path:
          workspace.id === 1
            ? paths.modelList()
            : paths.workspaceDetails(workspace.id, WorkspaceDetailsTab.ModelRegistry),
      },
    );
  }
  pageBreadcrumb.push({
    breadcrumbName: `${model.name} (${model.id})`,
    path: paths.modelDetails(model.id.toString()),
  });

  return (
    <Page
      breadcrumb={pageBreadcrumb}
      containerRef={pageRef}
      docTitle="Model Details"
      headerComponent={
        <ModelHeader
          fetchModel={fetchModel}
          model={model}
          workspace={workspace || undefined}
          onSwitchArchive={switchArchive}
          onUpdateTags={saveModelTags}
        />
      }
      id="modelDetails"
      notFound={pageError && isNotFound(pageError)}>
      <div className={css.base}>
        {modelVersions?.length === 0 ? (
          <div className={css.noVersions}>
            <p className={css.header}>No Model Versions</p>
            <p className={css.subtext}>
              Register a checkpoint from an experiment to add it to this model
            </p>
          </div>
        ) : (
          <InteractiveTable<ModelVersion, Settings>
            columns={columns}
            containerRef={pageRef}
            ContextMenu={actionDropdown}
            dataSource={modelVersions}
            loading={tableIsLoading}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="version"
            settings={settings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings}
            onChange={handleTableChange}
          />
        )}
        <Notes
          disabled={model.archived || !canModifyModel({ model: model })}
          disableTitle
          notes={{ contents: model.notes ?? '', name: 'Notes' }}
          onError={handleError}
          onSave={saveNotes}
        />
        <MetadataCard
          disabled={model.archived || !canModifyModel({ model: model })}
          metadata={model.metadata}
          onSave={saveMetadata}
        />
      </div>
    </Page>
  );
};

export default ModelDetails;
