import { Breadcrumb, Card, Tabs } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import InfoBox from 'components/InfoBox';
import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import MetadataCard from 'components/Metadata/MetadataCard';
import NotesCard from 'components/NotesCard';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { deleteModelVersion, getModelVersion, patchModelVersion } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize, getBatchNumber } from 'utils/workload';

import css from './ModelVersionDetails.module.scss';
import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';

const { TabPane } = Tabs;

interface Params {
  modelId: string;
  tab?: TabType;
  versionId: string;
}

enum TabType {
  CheckpointDetails = 'checkpoint-details',
  Overview = 'overview',
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Overview;

const ModelVersionDetails: React.FC = () => {
  const [ modelVersion, setModelVersion ] = useState<ModelVersion>();
  const { modelId, versionId, tab } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();
  const history = useHistory();
  const [ tabKey, setTabKey ] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);

  const basePath = paths.modelVersionDetails(modelId, versionId);

  const fetchModelVersion = useCallback(async () => {
    try {
      const versionData = await getModelVersion(
        { modelId: parseInt(modelId), versionId: parseInt(versionId) },
      );
      setModelVersion(prev => !isEqual(versionData, modelVersion) ? versionData : prev);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [ modelId, modelVersion, pageError, versionId ]);

  usePolling(fetchModelVersion);

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(`${basePath}/${key}`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      history.replace(`${basePath}/${tabKey}`);
    }
  }, [ basePath, history, tab, tabKey ]);

  const saveMetadata = useCallback(async (editedMetadata) => {
    try {
      await patchModelVersion({
        body: { id: parseInt(modelId), metadata: editedMetadata },
        modelId: parseInt(modelId),
        versionId: parseInt(versionId),
      });
      await fetchModelVersion();
    } catch (e) {
      handleError({
        message: 'Unable to save metadata.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ fetchModelVersion, modelId, versionId ]);

  const saveNotes = useCallback(async (editedNotes: string) => {
    try {
      const versionResponse = await patchModelVersion({
        body: { id: parseInt(modelId), notes: editedNotes },
        modelId: parseInt(modelId),
        versionId: parseInt(versionId),
      });
      setModelVersion(versionResponse);
    } catch (e) {
      handleError({
        message: 'Unable to update notes.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelId, versionId ]);

  const saveDescription = useCallback(async (editedDescription: string) => {
    try {
      await patchModelVersion({
        body: { comment: editedDescription, id: parseInt(modelId) },
        modelId: parseInt(modelId),
        versionId: parseInt(versionId),
      });
    } catch (e) {
      handleError({
        message: 'Unable to save description.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelId, versionId ]);

  const saveName = useCallback(async (editedName: string) => {
    try {
      await patchModelVersion({
        body: { id: parseInt(modelId), name: editedName },
        modelId: parseInt(modelId),
        versionId: parseInt(versionId),
      });
    } catch (e) {
      handleError({
        message: 'Unable to save name.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelId, versionId ]);

  const saveVersionTags = useCallback(async (newTags) => {
    try {
      await patchModelVersion({
        body: { id: parseInt(modelId), labels: newTags },
        modelId: parseInt(modelId),
        versionId: parseInt(versionId),
      });
      fetchModelVersion();
    } catch (e) {
      handleError({
        message: 'Unable to save tags.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ fetchModelVersion, modelId, versionId ]);

  const deleteVersion = useCallback(() => {
    deleteModelVersion({ modelId: modelVersion?.model.id ?? 0, versionId: modelVersion?.id ?? 0 });
    history.push(`/det/models/${modelVersion?.model.id}`);
  }, [ history, modelVersion?.id, modelVersion?.model.id ]);

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
          <Breadcrumb.Item>
            <Link path={paths.experimentDetails(modelVersion.checkpoint.experimentId || '')}>
              Experiment {modelVersion.checkpoint.experimentId}
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            <Link path={paths.trialDetails(
              modelVersion.checkpoint.trialId,
              modelVersion.checkpoint.experimentId,
            )}>
              Trial {modelVersion.checkpoint.trialId}
            </Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>Batch {totalBatchesProcessed}</Breadcrumb.Item>
        </Breadcrumb>,
        label: 'Source',
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
        onDeregisterVersion={deleteVersion}
        onSaveDescription={saveDescription}
        onSaveName={saveName}
        onUpdateTags={saveVersionTags} />}
      id="modelDetails">
      <Tabs
        defaultActiveKey="overview"
        style={{ height: 'auto' }}
        tabBarStyle={{ backgroundColor: 'var(--theme-colors-monochrome-17)', paddingLeft: 36 }}
        onChange={handleTabChange}>
        <TabPane key="model" tab="Model">
          <div className={css.base}>
            <Card title="Model Checkpoint">
              <InfoBox rows={checkpointInfo} separator />
            </Card>
            <Card title="Validation Metrics">
              <InfoBox rows={validationMetrics} separator />
            </Card>
            <MetadataCard
              metadata={modelVersion.metadata}
              onSave={saveMetadata} />
          </div>
        </TabPane>
        <TabPane key="notes" tab="Notes">
          <div className={css.base}>
            <NotesCard
              notes={modelVersion.notes ?? ''}
              onSave={saveNotes} />
          </div>
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default ModelVersionDetails;
