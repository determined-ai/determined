import { EditOutlined, SaveOutlined } from '@ant-design/icons';
import { Card } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import React, { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import EditableMetadata from 'components/EditableMetadata';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Spinner from 'components/Spinner';
import { modelVersionNameRenderer, modelVersionNumberRenderer,
  relativeTimeRenderer } from 'components/Table';
import usePolling from 'hooks/usePolling';
import { getModelDetails } from 'services/api';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion, ModelVersions } from 'types';
import { isEqual } from 'utils/data';

import ModelHeader from './ModelDetails/ModelHeader';

interface Params {
  modelId: string;
}

const ModelDetails: React.FC = () => {
  const [ model, setModel ] = useState<ModelVersions>();
  const { modelId } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();
  const [ editingMetadata, setEditingMetadata ] = useState(false);
  const [ editedMetadata, setEditedMetadata ] = useState<Record<string, string>>({});

  const id = parseInt(modelId);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModelDetails(
        { modelName: 'mnist-prod', sortBy: 'SORT_BY_VERSION' },
      );
      if (!isEqual(modelData, model)) setModel(modelData);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [ model, pageError ]);

  usePolling(fetchModel);

  const columns = useMemo(() => {
    const tableColumns: ColumnsType<ModelVersion> = [
      {
        dataIndex: 'version',
        key: V1GetModelVersionsRequestSortBy.VERSION,
        render: modelVersionNumberRenderer,
        sorter: true,
        title: 'Version',
        width: 1,
      },
      {
        dataIndex: 'name',
        render: modelVersionNameRenderer,
        title: 'Name',
      },
      {
        dataIndex: 'description',
        title: 'Description',
      },
      {
        dataIndex: 'lastUpdatedTime',
        render: relativeTimeRenderer,
        sorter: true,
        title: 'Last updated',
      },
      { dataIndex: 'tags', title: 'Tags' },
    ];

    return tableColumns;
  }, []);

  const metadata = useMemo(() => {
    return Object.entries(model?.model.metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ model?.model.metadata ]);

  const editMetadata = useCallback(() => {
    setEditingMetadata(true);
  }, []);

  const saveMetadata = useCallback(() => {
    setEditingMetadata(false);
    // patchModel with editedMetadata
  }, []);

  const switchArchive = useCallback(() => {
    //check current archive status, switch it
  }, []);

  const deleteModel = useCallback(() => {
    //delete model, take user to model registry page
  }, []);

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
        archived={false}
        model={model.model}
        onAddMetadata={editMetadata}
        onDelete={deleteModel}
        onSwitchArchive={switchArchive} />}
      id="modelDetails">
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        marginLeft: 20,
        marginRight: 20,
      }}>{
          model.modelVersions.length === 0 ?
            <div style={{
              alignItems: 'center',
              display: 'flex',
              flexDirection: 'column',
              margin: 'var(--theme-sizes-layout-huge)',
            }}>
              <p>No Model Versions</p>
              <p style={{
                color: 'var(--theme-colors-monochrome-9)',
                fontSize: 'var(--theme-sizes-font-medium',
                maxWidth: '370px',
                textAlign: 'center',
              }}>
                Register a checkpoint from an experiment to add it to this model
              </p>
            </div> :
            <ResponsiveTable
              columns={columns}
              dataSource={model.modelVersions}
              pagination={{ hideOnSinglePage: true }}
              showSorterTooltip={false}
            />
        }
        {metadata.length > 0 || editingMetadata &&
          <Card
            extra={editingMetadata ?
              <SaveOutlined onClick={saveMetadata} /> :
              <EditOutlined onClick={editMetadata} />}
            title={'Metadata'}>
            <EditableMetadata
              editing={editingMetadata}
              metadata={model?.model.metadata}
              updateMetadata={setEditedMetadata} />
          </Card>
        }
      </div>
    </Page>
  );
};

export default ModelDetails;
