import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Dropdown, Menu, Modal, Space, Tooltip } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import EditableMetadata from 'components/EditableMetadata';
import Icon from 'components/Icon';
import IconButton from 'components/IconButton';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Spinner from 'components/Spinner';
import { modelVersionNameRenderer, modelVersionNumberRenderer,
  relativeTimeRenderer, userRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import usePolling from 'hooks/usePolling';
import { archiveModel, deleteModel, deleteModelVersion, getModelDetails, patchModel,
  patchModelVersion, unarchiveModel } from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion, ModelVersions } from 'types';
import { isEqual } from 'utils/data';

import css from './ModelDetails.module.scss';
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
  const [ isEditingMetadata, setIsEditingMetadata ] = useState(false);
  const [ editedMetadata, setEditedMetadata ] = useState<Record<string, string>>({});
  const history = useHistory();

  const id = parseInt(modelId);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModelDetails(
        { modelId: id, sortBy: 'SORT_BY_VERSION' },
      );
      if (!isEqual(modelData, model)) setModel(modelData);
      setIsLoading(false);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
      setIsLoading(false);
    }
  }, [ id, model, pageError ]);

  usePolling(fetchModel);

  useEffect(() => {
    fetchModel();
    setIsLoading(true);
  }, [ fetchModel ]);

  const deleteVersion = useCallback((version: ModelVersion) => {
    deleteModelVersion({ modelId: version.model.id, versionId: version.id });
    fetchModel();
    setIsLoading(true);
  }, [ fetchModel ]);

  const downloadVersion = useCallback((version: ModelVersion) => {
    //open download popover
    //eslint-disable-next-line
    console.log(version)
  }, []);

  const setModelVersionTags = useCallback((modelId, versionId, tags) => {
    patchModelVersion({ body: { id: versionId, labels: tags }, modelId, versionId });
    fetchModel();
    setIsLoading(true);
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
        onChange={(tags) => setModelVersionTags(record.model.id, record.id, tags)}
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
                onClick={() => showConfirmDelete(record)}>
                  Delete Version
              </Menu.Item>
            </Menu>
          )}>
          <Button className={css.overflow} type="text">
            <Icon name="overflow-vertical" size="tiny" />
          </Button>
        </Dropdown>
      );
    };

    const actionRenderer = (_:string, record: ModelVersion) => {
      return <div className={css.center}>
        <IconButton
          icon="download"
          iconSize="large"
          label="Download Model"
          type="text"
          onClick={() => downloadVersion(record)} />
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
        dataIndex: 'description',
        title: 'Description',
      },
      {
        dataIndex: 'lastUpdatedTime',
        render: (date: Date, record: ModelVersion) =>
          relativeTimeRenderer(date ?? record.creationTime),
        sorter: true,
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

    return tableColumns;
  }, [ showConfirmDelete, downloadVersion, model?.model.username, setModelVersionTags, user ]);

  const metadata = useMemo(() => {
    return Object.entries(model?.model.metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ model?.model.metadata ]);

  const editMetadata = useCallback(() => {
    setIsEditingMetadata(true);
  }, []);

  const saveMetadata = useCallback(() => {
    setIsEditingMetadata(false);
    patchModel({
      body: { id: parseInt(modelId), metadata: editedMetadata },
      modelId: parseInt(modelId),
    });
    fetchModel();
  }, [ editedMetadata, fetchModel, modelId ]);

  const cancelEditMetadata = useCallback(() => {
    setIsEditingMetadata(false);
  }, []);

  const saveDescription = useCallback(async (editedDescription: string) => {
    await patchModel({
      body: { description: editedDescription, id: parseInt(modelId) },
      modelId: parseInt(modelId),
    });
  }, [ modelId ]);

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
        onSwitchArchive={switchArchive} />}
      id="modelDetails">
      <div className={css.base}>{
        model.modelVersions.length === 0 ?
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
            pagination={{ hideOnSinglePage: true }}
            showSorterTooltip={false}
            size="small"
          />
      }
      {(metadata.length > 0 || isEditingMetadata) &&
          <Card
            extra={isEditingMetadata ? (
              <Space size="small">
                <Button size="small" onClick={cancelEditMetadata}>Cancel</Button>
                <Button size="small" type="primary" onClick={saveMetadata}>Save</Button>
              </Space>
            ) : (
              <Tooltip title="Edit">
                <EditOutlined onClick={editMetadata} />
              </Tooltip>
            )}
            title={'Metadata'}>
            <EditableMetadata
              editing={isEditingMetadata}
              metadata={model?.model.metadata}
              updateMetadata={setEditedMetadata} />
          </Card>
      }
      </div>
    </Page>
  );
};

export default ModelDetails;
