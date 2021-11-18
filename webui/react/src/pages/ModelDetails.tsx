import { Button, Dropdown, Menu, Modal } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import { SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import DownloadModelPopover from 'components/DownloadModelPopover';
import Icon from 'components/Icon';
import IconButton from 'components/IconButton';
import Message, { MessageType } from 'components/Message';
import MetadataCard from 'components/MetadataCard';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Spinner from 'components/Spinner';
import { getFullPaginationConfig, modelVersionNameRenderer, modelVersionNumberRenderer,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { archiveModel, deleteModel, deleteModelVersion, getModelDetails, patchModel,
  patchModelVersion, unarchiveModel } from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import { isAborted, isNotFound, validateDetApiEnum } from 'services/utils';
import { ModelVersion, ModelVersions } from 'types';
import { isEqual } from 'utils/data';

import css from './ModelDetails.module.scss';
import settingsConfig, { Settings } from './ModelDetails/ModelDetails.settings';
import ModelHeader from './ModelDetails/ModelHeader';

interface Params {
  modelId: string;
}

const ModelDetails: React.FC = () => {
  const { auth: { user } } = useStore();
  const [ model, setModel ] = useState<ModelVersions>();
  const { modelId } = useParams<Params>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ pageError, setPageError ] = useState<Error>();
  const [ forceEditMetadata, setForceEditMetadata ] = useState(false);
  const [ total, setTotal ] = useState(0);
  const history = useHistory();

  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const id = parseInt(modelId);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModelDetails(
        {
          limit: settings.tableLimit,
          modelId: id,
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
  }, [ id, pageError, settings ]);

  usePolling(fetchModel);

  useEffect(() => {
    setIsLoading(true);
    fetchModel();
  }, [ fetchModel ]);

  const deleteVersion = useCallback(async (version: ModelVersion) => {
    try {
      setIsLoading(true);
      await deleteModelVersion({ modelId: version.model.id, versionId: version.id });
      await fetchModel();
    } catch (e) {
      handleError({
        message: `Unable to delete model version ${version.id}.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModel ]);

  const saveModelVersionTags = useCallback(async (modelId, versionId, tags) => {
    try {
      setIsLoading(true);
      await patchModelVersion({ body: { id: versionId, labels: tags }, modelId, versionId });
      await fetchModel();
    } catch (e) {
      handleError({
        message: `Unable to update model version ${versionId} tags.`,
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

  const columns = useMemo(() => {
    const labelsRenderer = (value: string, record: ModelVersion) => (
      <TagList
        compact
        tags={record.labels ?? []}
        onChange={(tags) => saveModelVersionTags(record.model.id, record.id, tags)}
      />
    );

    const overflowRenderer = (_:string, record: ModelVersion) => {
      const isDeletable = user?.isAdmin
        || user?.username === model?.model.username
        || user?.username === record.username;
      return (
        <Dropdown
          overlay={(
            <Menu>
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

    const actionRenderer = (_:string, record: ModelVersion) => {
      return <div className={css.center}>
        <DownloadModelPopover modelVersion={record}>
          <IconButton
            icon="download"
            iconSize="large"
            label="Download Model"
            type="text" />
        </DownloadModelPopover>
      </div>;
    };

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
      { dataIndex: 'labels', render: labelsRenderer, title: 'Tags', width: 120 },
      { render: actionRenderer, title: 'Actions', width: 1 },
      { render: overflowRenderer, title: '', width: 1 },
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
    settings.sortDesc ]);

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

  const editMetadata = useCallback(() => {
    setForceEditMetadata(true);
  }, []);

  const saveMetadata = useCallback(async (editedMetadata) => {
    try {
      await patchModel({
        body: { id: parseInt(modelId), metadata: editedMetadata },
        modelId: parseInt(modelId),
      });
      await fetchModel();
    } catch (e) {
      handleError({
        message: 'Unable to save metadata.',
        silent: true,
        type: ErrorType.Api,
      });
    }

  }, [ fetchModel, modelId ]);

  const saveDescription = useCallback(async (editedDescription: string) => {
    try {
      await patchModel({
        body: { description: editedDescription, id: parseInt(modelId) },
        modelId: parseInt(modelId),
      });
    } catch (e) {
      handleError({
        message: 'Unable to save description.',
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ modelId ]);

  const saveName = useCallback(async (editedName: string) => {
    try {
      await patchModel({
        body: { id: parseInt(modelId), name: editedName },
        modelId: parseInt(modelId),
      });
    } catch (e) {
      handleError({
        message: 'Unable to save name.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelId ]);

  const saveModelTags = useCallback(async (editedTags) => {
    try {
      await patchModel({
        body: { id: parseInt(modelId), labels: editedTags },
        modelId: parseInt(modelId),
      });
      fetchModel();
    } catch (e) {
      handleError({
        message: 'Unable to update model tags.',
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ fetchModel, modelId ]);

  const switchArchive = useCallback(() => {
    if (model?.model.archived) {
      unarchiveModel({ modelId: parseInt(modelId) });
    } else {
      archiveModel({ modelId: parseInt(modelId) });
    }
  }, [ model?.model.archived, modelId ]);

  const deleteCurrentModel = useCallback(() => {
    deleteModel({ modelId: parseInt(modelId) });
    history.push('/det/models');
  }, [ history, modelId ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Model ID ${modelId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelId}` :
      `Unable to fetch model ${modelId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!model) {
    return <Spinner tip={`Loading model ${modelId} details...`} />;
  }

  return (
    <Page
      docTitle="Model Details"
      headerComponent={<ModelHeader
        model={model.model}
        onAddMetadata={editMetadata}
        onDelete={deleteCurrentModel}
        onSaveDescription={saveDescription}
        onSaveName={saveName}
        onSwitchArchive={switchArchive}
        onUpdateTags={saveModelTags} />}
      id="modelDetails">
      <div className={css.base}>
        {model.modelVersions.length === 0 ?
          <div className={css.noVersions}>
            <p>No Model Versions</p>
            <p className={css.subtext}>
                Register a checkpoint from an experiment to add it to this model
            </p>
          </div> :
          <ResponsiveTable
            columns={columns}
            dataSource={model.modelVersions}
            loading={isLoading}
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
            showSorterTooltip={false}
            size="small"
            onChange={handleTableChange}
          />
        }
        <MetadataCard
          forceEdit={forceEditMetadata}
          metadata={model.model.metadata}
          onSave={saveMetadata} />
      </div>
    </Page>
  );
};

export default ModelDetails;
