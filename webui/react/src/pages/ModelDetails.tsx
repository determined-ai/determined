import { SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import InlineEditor from 'components/InlineEditor';
import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
import MetadataCard from 'components/Metadata/MetadataCard';
import NotesCard from 'components/NotesCard';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import { defaultRowClassName, getFullPaginationConfig,
  modelVersionNameRenderer, modelVersionNumberRenderer,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import useModalModelDownload from 'hooks/useModal/Model/useModalModelDownload';
import useModalModelVersionDelete from 'hooks/useModal/Model/useModalModelVersionDelete';
import usePolling from 'hooks/usePolling';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import {
  archiveModel, getModelDetails, patchModel,
  patchModelVersion, unarchiveModel,
} from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { isAborted, isNotFound, validateDetApiEnum } from 'shared/utils/service';
import { ModelVersion, ModelVersions } from 'types';
import handleError from 'utils/error';

import css from './ModelDetails.module.scss';
import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  Settings,
} from './ModelDetails/ModelDetails.settings';
import ModelHeader from './ModelDetails/ModelHeader';
import ModelVersionActionDropdown from './ModelDetails/ModelVersionActionDropdown';

interface Params {
  modelId: string;
}

const ModelDetails: React.FC = () => {
  const [ model, setModel ] = useState<ModelVersions>();
  const modelId = decodeURIComponent(useParams<Params>().modelId);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ pageError, setPageError ] = useState<Error>();
  const [ total, setTotal ] = useState(0);
  const pageRef = useRef<HTMLElement>(null);

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModelDetails(
        {
          limit: settings.tableLimit,
          modelName: modelId,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetModelVersionsRequestSortBy, settings.sortKey),
        },
      );
      setTotal(modelData?.pagination.total || 0);
      setModel((prev) => !isEqual(modelData, prev) ? modelData : prev);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
    setIsLoading(false);
  }, [ modelId, pageError, settings ]);

  const {
    contextHolder: modalModelDownloadContextHolder,
    modalOpen: openModelDownload,
  } = useModalModelDownload();

  const {
    contextHolder: modalModelVersionDeleteContextHolder,
    modalOpen: openModelVersionDelete,
  } = useModalModelVersionDelete();

  usePolling(fetchModel, { rerunOnNewFn: true });

  useEffect(() => {
    setIsLoading(true);
    fetchModel();
  }, [ fetchModel ]);

  const downloadModel = useCallback((version: ModelVersion) => {
    openModelDownload(version);
  }, [ openModelDownload ]);

  const deleteModelVersion = useCallback((version: ModelVersion) => {
    openModelVersionDelete(version);
  }, [ openModelVersionDelete ]);

  const saveModelVersionTags = useCallback(async (modelName, versionId, tags) => {
    try {
      setIsLoading(true);
      await patchModelVersion({ body: { labels: tags, modelName }, modelName, versionId });
      await fetchModel();
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to update model version ${versionId} tags.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModel ]);

  const saveVersionDescription = useCallback(async (
    editedDescription: string,
    versionId: number,
  ) => {
    try {
      const modelName = model?.model.name;
      if (modelName) {
        await patchModelVersion({
          body: { comment: editedDescription, modelName },
          modelName,
          versionId,
        });
      }
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save version description.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [ model?.model.name ]);

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ModelVersion) => (
      <TagList
        compact
        disabled={record.model.archived}
        tags={record.labels ?? []}
        onChange={(tags) => saveModelVersionTags(record.model.name, record.id, tags)}
      />
    );

    const actionRenderer = (_:string, record: ModelVersion) => (
      <ModelVersionActionDropdown
        onDelete={() => deleteModelVersion(record)}
        onDownload={() => downloadModel(record)}
      />
    );

    const descriptionRenderer = (value:string, record: ModelVersion) => (
      <InlineEditor
        disabled={record.model.archived}
        placeholder={record.model.archived ? 'Archived' : 'Add description...'}
        value={record.comment ?? ''}
        onSave={(newDescription: string) => saveVersionDescription(newDescription, record.id)}
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
        render: userRenderer,
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
  ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ModelVersion>;
    if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

    const newSettings = {
      sortDesc: order === 'descend',
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      sortKey: columnKey as any,
      tableLimit: tablePagination.pageSize,
      tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
    };
    const shouldPush = settings.tableOffset !== newSettings.tableOffset;
    updateSettings(newSettings, shouldPush);
  }, [ columns, settings.tableOffset, updateSettings ]);

  const saveMetadata = useCallback(async (editedMetadata) => {
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

  }, [ fetchModel, model?.model.name ]);

  const saveDescription = useCallback(async (editedDescription: string) => {
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
      setIsLoading(false);
    }
  }, [ model?.model.name ]);

  const saveName = useCallback(async (editedName: string) => {
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
  }, [ model?.model.name ]);

  const saveNotes = useCallback(async (editedNotes: string) => {
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
  }, [ model?.model.name, fetchModel ]);

  const saveModelTags = useCallback(async (editedTags) => {
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
      setIsLoading(false);
    }
  }, [ fetchModel, model?.model.name ]);

  const switchArchive = useCallback(() => {
    const modelName = model?.model.name;
    if (modelName) {
      if (model?.model.archived) {
        unarchiveModel({ modelName });
      } else {
        archiveModel({ modelName });
      }
    }
  }, [ model?.model.archived, model?.model.name ]);

  const actionDropdown = useCallback(({ record, onVisibleChange, children }) => (
    <ModelVersionActionDropdown
      trigger={[ 'contextMenu' ]}
      onDelete={() => deleteModelVersion(record)}
      onDownload={() => downloadModel(record)}
      onVisibleChange={onVisibleChange}>
      {children}
    </ModelVersionActionDropdown>
  ), [ deleteModelVersion, downloadModel ]);

  if (!modelId) {
    return <Message title="Model name is empty" />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message =
      `Unable to fetch model ${modelId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!model) {
    return <Spinner tip={`Loading model ${modelId} details...`} />;
  }

  return (
    <Page
      containerRef={pageRef}
      docTitle="Model Details"
      headerComponent={(
        <ModelHeader
          model={model.model}
          onSaveDescription={saveDescription}
          onSaveName={saveName}
          onSwitchArchive={switchArchive}
          onUpdateTags={saveModelTags}
        />
      )}
      id="modelDetails">
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
            loading={isLoading}
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="version"
            settings={settings as InteractiveTableSettings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
            onChange={handleTableChange}
          />
        )}
        <NotesCard
          disabled={model.model.archived}
          notes={model.model.notes ?? ''}
          onSave={saveNotes}
        />
        <MetadataCard
          disabled={model.model.archived}
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
