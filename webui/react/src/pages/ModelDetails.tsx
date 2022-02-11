import { Button, Dropdown, Menu, Modal } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import { SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import DownloadModelModal from 'components/DownloadModelModal';
import Icon from 'components/Icon';
import InlineEditor from 'components/InlineEditor';
import Message, { MessageType } from 'components/Message';
import MetadataCard from 'components/Metadata/MetadataCard';
import NotesCard from 'components/NotesCard';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Spinner from 'components/Spinner';
import { getFullPaginationConfig, modelVersionNameRenderer, modelVersionNumberRenderer,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { paths, routeToReactUrl } from 'routes/utils';
import { archiveModel, deleteModel, deleteModelVersion, getModelDetails, patchModel,
  patchModelVersion, unarchiveModel } from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import { isAborted, isNotFound, validateDetApiEnum } from 'services/utils';
import { ModelVersion, ModelVersions } from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';

import css from './ModelDetails.module.scss';
import settingsConfig, { Settings } from './ModelDetails/ModelDetails.settings';
import ModelHeader from './ModelDetails/ModelHeader';

interface Params {
  modelName: string;
}

const ModelDetails: React.FC = () => {
  const { auth: { user } } = useStore();
  const [ model, setModel ] = useState<ModelVersions>();
  const modelName = decodeURIComponent(useParams<Params>().modelName);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ pageError, setPageError ] = useState<Error>();
  const [ total, setTotal ] = useState(0);
  const history = useHistory();

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModelDetails(
        {
          limit: settings.tableLimit,
          modelName,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetModelVersionsRequestSortBy, settings.sortKey),
        },
      );
      setTotal(modelData?.pagination.total || 0);
      setModel(prev => !isEqual(modelData, prev) ? modelData : prev);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
    setIsLoading(false);
  }, [ modelName, pageError, settings ]);

  usePolling(fetchModel);

  useEffect(() => {
    setIsLoading(true);
    fetchModel();
  }, [ fetchModel ]);

  const deleteVersion = useCallback(async (version: ModelVersion) => {
    try {
      setIsLoading(true);
      await deleteModelVersion({ modelName: version.model.name, versionId: version.id });
      await fetchModel();
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to delete model version ${version.id}.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModel ]);

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

  const showConfirmDelete = useCallback((version: ModelVersion) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this version "Version ${version.version}"
      from this model?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Version',
      okType: 'danger',
      onOk: () => deleteVersion(version),
      title: 'Confirm Delete',
    });
  }, [ deleteVersion ]);

  const saveVersionDescription =
    useCallback(async (editedDescription: string, versionId: number) => {
      try {
        await patchModelVersion({
          body: { comment: editedDescription, modelName },
          modelName,
          versionId,
        });
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save version description.',
          silent: false,
          type: ErrorType.Api,
        });
        return e;
      }
    }, [ modelName ]);

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ModelVersion) => (
      <TagList
        compact
        disabled={record.model.archived}
        tags={record.labels ?? []}
        onChange={(tags) => saveModelVersionTags(record.model.name, record.id, tags)}
      />
    );

    const OverflowRenderer = (_:string, record: ModelVersion) => {
      const isDeletable = user?.isAdmin
        || user?.username === model?.model.username
        || user?.username === record.username;
      return (
        <Dropdown
          overlay={(
            <Menu>
              {useActionRenderer(_, record)}
              <Menu.Item
                danger
                disabled={!isDeletable}
                key="delete-version"
                onClick={() => showConfirmDelete(record)}>
                Delete Version
              </Menu.Item>
            </Menu>
          )}
          trigger={[ 'click' ]}>
          <Button className={css.overflow} type="text">
            <Icon name="overflow-vertical" size="tiny" />
          </Button>
        </Dropdown>
      );
    };

    const descriptionRenderer = (value:string, record: ModelVersion) => (
      <InlineEditor
        disabled={record.model.archived}
        placeholder="Add description..."
        value={value}
        onSave={(newDescription: string) => saveVersionDescription(newDescription, record.id)}
      />
    );

    const tableColumns: ColumnsType<ModelVersion> = [
      {
        dataIndex: 'version',
        key: V1GetModelVersionsRequestSortBy.VERSION,
        render: modelVersionNumberRenderer,
        sorter: true,
        title: 'V',
        width: 1,
      },
      {
        dataIndex: 'name',
        render: modelVersionNameRenderer,
        title: 'Name',
        width: 250,
      },
      {
        dataIndex: 'comment',
        render: descriptionRenderer,
        title: 'Description',
      },
      {
        dataIndex: 'lastUpdatedTime',
        render: (date: Date, record: ModelVersion) =>
          relativeTimeRenderer(date ?? record.creationTime),
        title: 'Last updated',
        width: 140,
      },
      {
        dataIndex: 'username',
        render: (username: string, record: ModelVersion, index) =>
          username ?
            userRenderer(username, record, index) :
            userRenderer(record.model.username, record.model, index),
        title: 'User',
        width: 1,
      },
      { dataIndex: 'labels', render: tagsRenderer, title: 'Tags', width: 120 },
      { render: OverflowRenderer, title: '', width: 1 },
    ];

    return tableColumns.map(column => {
      column.sortOrder = null;
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });
  }, [ showConfirmDelete,
    model?.model.username,
    saveModelVersionTags,
    user,
    settings.sortKey,
    settings.sortDesc,
    saveVersionDescription ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ModelVersion>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

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
      await patchModel({
        body: { metadata: editedMetadata, name: modelName },
        modelName,
      });
      await fetchModel();
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save metadata.',
        silent: false,
        type: ErrorType.Api,
      });
    }

  }, [ fetchModel, modelName ]);

  const saveDescription = useCallback(async (editedDescription: string) => {
    try {
      await patchModel({
        body: { description: editedDescription, name: modelName },
        modelName,
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save description.',
        silent: false,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ modelName ]);

  const saveName = useCallback(async (editedName: string) => {
    try {
      await patchModel({
        body: { name: editedName },
        modelName,
      });
      routeToReactUrl(paths.modelDetails(editedName));
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save name.',
        silent: false,
        type: ErrorType.Api,
      });
      return e;
    }
  }, [ modelName ]);

  const saveNotes = useCallback(async (editedNotes: string) => {
    try {
      await patchModel({
        body: { name: modelName, notes: editedNotes },
        modelName,
      });
      await fetchModel();
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to update notes.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelName, fetchModel ]);

  const saveModelTags = useCallback(async (editedTags) => {
    try {
      await patchModel({
        body: { labels: editedTags, name: modelName },
        modelName,
      });
      fetchModel();
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to update model tags.',
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModel, modelName ]);

  const switchArchive = useCallback(() => {
    if (model?.model.archived) {
      unarchiveModel({ modelName });
    } else {
      archiveModel({ modelName });
    }
  }, [ model?.model.archived, modelName ]);

  const deleteCurrentModel = useCallback(() => {
    deleteModel({ modelName });
    history.push('/det/models');
  }, [ history, modelName ]);

  if (!modelName) {
    return <Message title="Model name is empty" />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelName}` :
      `Unable to fetch model ${modelName}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!model) {
    return <Spinner tip={`Loading model ${modelName} details...`} />;
  }

  return (
    <Page
      docTitle="Model Details"
      headerComponent={(
        <ModelHeader
          model={model.model}
          onDelete={deleteCurrentModel}
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
          <ResponsiveTable
            columns={columns}
            dataSource={model.modelVersions}
            loading={isLoading}
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
            rowKey="id"
            showSorterTooltip={false}
            size="small"
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
    </Page>
  );
};

const useActionRenderer = (_:string, record: ModelVersion) => {
  const [ showModal, setShowModal ] = useState(false);

  return (
    <>
      <Menu.Item
        key="download"
        onClick={() => setShowModal(true)}>
        Download
      </Menu.Item>
      <DownloadModelModal
        modelVersion={record}
        visible={showModal}
        onClose={() => setShowModal(false)}
      />
    </>
  );
};

export default ModelDetails;
