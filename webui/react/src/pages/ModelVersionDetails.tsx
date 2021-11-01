import { CopyOutlined, EditOutlined } from '@ant-design/icons';
import { Breadcrumb, Button, Card, Space, Tabs, Tooltip } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import EditableMetadata from 'components/EditableMetadata';
import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Message, { MessageType } from 'components/Message';
import NotesCard from 'components/NotesCard';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getModelVersion, patchModelVersion } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize, getBatchNumber } from 'utils/types';

import css from './ModelVersionDetails.module.scss';
import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';

const { TabPane } = Tabs;

interface Params {
  modelId: string;
  versionId: string;
}

const ModelVersionDetails: React.FC = () => {
  const [ modelVersion, setModelVersion ] = useState<ModelVersion>();
  const { modelId, versionId } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();
  const [ isEditingMetadata, setIsEditingMetadata ] = useState(false);
  const [ editedMetadata, setEditedMetadata ] = useState<Record<string, string>>({});

  const fetchModelVersion = useCallback(async () => {
    try {
      const versionData = await getModelVersion(
        { modelId: parseInt(modelId), versionId: parseInt(versionId) },
      );
      if (!isEqual(versionData, modelVersion)) setModelVersion(versionData);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [ modelId, modelVersion, pageError, versionId ]);

  usePolling(fetchModelVersion);

  const referenceText = useMemo(() => {
    return (
      `from determined.experimental import Determined
model = Determined.getModel("${modelVersion?.model?.name}")
ckpt = model.get_version("${modelVersion?.version}")
ckpt_path = ckpt.download()
ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))

# WARNING: From here on out, this might not be possible to automate. Requires research.
from model import build_model
model = build_model()
model.load_state_dict(ckpt['models_state_dict'][0])

# If you get this far, you should be able to run \`model.eval()\``);
  }, [ modelVersion ]);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(referenceText);
  }, [ referenceText ]);

  const metadata = Object.entries(modelVersion?.metadata ?? {}).map((pair) => {
    return ({ content: pair[1], label: pair[0] } as InfoRow);
  });

  const editMetadata = useCallback(() => {
    setIsEditingMetadata(true);
  }, []);

  const saveMetadata = useCallback(() => {
    setIsEditingMetadata(false);
    patchModelVersion({
      body: { id: parseInt(modelId), metadata: editedMetadata },
      modelId: parseInt(modelId),
      versionId: parseInt(versionId),
    });
    fetchModelVersion();
  }, [ editedMetadata, fetchModelVersion, modelId, versionId ]);

  const cancelEditMetadata = useCallback(() => {
    setIsEditingMetadata(false);
  }, []);

  const saveNotes = useCallback((editedNotes: string) => {
    patchModelVersion({
      body: { id: parseInt(modelId), notes: editedNotes },
      modelId: parseInt(modelId),
      versionId: parseInt(versionId),
    });
  }, [ modelId, versionId ]);

  const deleteVersion = useCallback(() => {
    //delete/deregister version
  }, []);

  const renderResource = (resource: string, size: string): React.ReactNode => {
    return (
      <div className={css.resource} key={resource}>
        <div className={css.resourceName}>{resource}</div>
        <div className={css.resourceSpacer} />
        <div className={css.resourceSize}>{size}</div>
      </div>
    );
  };

  const checkpointInfo = useMemo(() => {
    if (!modelVersion?.checkpoint) return [];
    const checkpointResources = modelVersion.checkpoint.resources || {};
    const resources = Object.keys(modelVersion.checkpoint.resources || {})
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map(key => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));
    const totalBatchesProcessed = getBatchNumber(modelVersion.checkpoint);
    return [
      {
        content: <Breadcrumb className={css.link}>
          <Breadcrumb.Item
            className={css.link}
            href={paths.experimentDetails(modelVersion.checkpoint.experimentId || '')}>
            Experiment {modelVersion.checkpoint.experimentId}
          </Breadcrumb.Item>
          <Breadcrumb.Item
            className={css.link}
            href={paths.trialDetails(
              modelVersion.checkpoint.trialId,
              modelVersion.checkpoint.experimentId,
            )}>
          Trial {modelVersion.checkpoint.trialId}
          </Breadcrumb.Item>
          <Breadcrumb.Item className={css.link}>Batch {totalBatchesProcessed}</Breadcrumb.Item>
        </Breadcrumb>,
        label: 'Checkpoint',
      },
      { content: modelVersion.checkpoint.uuid, label: 'Checkpoint UUID' },
      {
        content: humanReadableBytes(checkpointSize(modelVersion.checkpoint)),
        label: 'Total Size',
      },
      {
        content: resources.map(resource => renderResource(resource.name, resource.size)),
        label: 'Code',
      } ];
  }, [ modelVersion?.checkpoint ]);

  const validationMetrics = useMemo(() => {
    if (!modelVersion?.checkpoint) return [];
    const metrics = Object.entries(modelVersion?.checkpoint.metrics?.validationMetrics || {});
    return metrics.map(metric => ({
      content: metric[1],
      label: metric[0],
    }));
  }, [ modelVersion?.checkpoint ]);

  if (isNaN(parseInt(modelId))) {
    return <Message title={`Invalid Model ID ${modelId}`} />;
  } else if (isNaN(parseInt(versionId))) {
    return <Message title={`Invalid Version ID ${versionId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelId} version ${versionId}` :
      `Unable to fetch model ${modelId} version ${versionId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!modelVersion) {
    return <Spinner tip={`Loading model ${modelId} version ${versionId} details...`} />;
  }

  return (
    <Page
      bodyNoPadding
      docTitle="Model Version Details"
      headerComponent={<ModelVersionHeader
        modelVersion={modelVersion}
        onAddMetadata={editMetadata}
        onDeregisterVersion={deleteVersion} />}
      id="modelDetails">
      <Tabs
        defaultActiveKey="1"
        tabBarStyle={{ backgroundColor: 'var(--theme-colors-monochrome-17)', paddingLeft: 36 }}>
        <TabPane key="1" tab="Overview">
          <div className={css.base}>
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
              metadata={modelVersion.metadata ?? {}}
              updateMetadata={setEditedMetadata} />
          </Card>
            }
            <NotesCard
              notes={modelVersion.notes ?? ''}
              style={{ height: 350 }}
              onSave={saveNotes} />
            <Card
              extra={(
                <Tooltip title="Copied!" trigger="click">
                  <CopyOutlined onClick={handleCopy} />
                </Tooltip>
              )}
              title={<>How to reference this model <Icon name="info" /></>}>
              <pre>{referenceText}</pre>
            </Card>
          </div>
        </TabPane>
        <TabPane key="2" tab="Checkpoint Details">
          <div className={css.base}>
            <Card title="Source">
              <InfoBox rows={checkpointInfo} seperator />
            </Card>
            <Card title="Validation Metrics">
              <InfoBox rows={validationMetrics} seperator />
            </Card>
          </div>
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default ModelVersionDetails;
