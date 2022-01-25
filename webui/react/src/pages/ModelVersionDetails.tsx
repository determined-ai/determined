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
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { deleteModelVersion, getModelVersion, patchModelVersion } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize, getBatchNumber } from 'utils/workload';

import css from './ModelVersionDetails.module.scss';
import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';

const { TabPane } = Tabs;

interface Params {
  modelName: string;
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
  const { modelName, versionId, tab } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();
  const history = useHistory();
  const [ tabKey, setTabKey ] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);

  const basePath = paths.modelVersionDetails(modelName, versionId);

  const fetchModelVersion = useCallback(async () => {
    try {
      const versionData = await getModelVersion(
        { modelName, versionId: parseInt(versionId) },
      );
      setModelVersion(prev => !isEqual(versionData, modelVersion) ? versionData : prev);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [ modelName, modelVersion, pageError, versionId ]);

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
        body: { metadata: editedMetadata, modelName },
        modelName,
        versionId: parseInt(versionId),
      });
      await fetchModelVersion();
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save metadata.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ fetchModelVersion, modelName, versionId ]);

  const saveNotes = useCallback(async (editedNotes: string) => {
    try {
      const versionResponse = await patchModelVersion({
        body: { modelName, notes: editedNotes },
        modelName,
        versionId: parseInt(versionId),
      });
      setModelVersion(versionResponse);
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to update notes.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelName, versionId ]);

  const saveDescription = useCallback(async (editedDescription: string) => {
    try {
      await patchModelVersion({
        body: { comment: editedDescription, modelName },
        modelName,
        versionId: parseInt(versionId),
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save description.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelName, versionId ]);

  const saveName = useCallback(async (editedName: string) => {
    try {
      await patchModelVersion({
        body: { modelName, name: editedName },
        modelName,
        versionId: parseInt(versionId),
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save name.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ modelName, versionId ]);

  const saveVersionTags = useCallback(async (newTags) => {
    try {
      await patchModelVersion({
        body: { labels: newTags, modelName },
        modelName,
        versionId: parseInt(versionId),
      });
      fetchModelVersion();
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to save tags.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ fetchModelVersion, modelName, versionId ]);

  const deleteVersion = useCallback(() => {
    deleteModelVersion({
      modelName: modelVersion?.model.name ?? '',
      versionId: modelVersion?.id ?? 0,
    });
    history.push(`/det/models/${modelVersion?.model.name}`);
  }, [ history, modelVersion?.id, modelVersion?.model.name ]);

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
        content: (
          <Breadcrumb className={css.link}>
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
          </Breadcrumb>
        ),
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

  if (!modelName) {
    return <Message title="Model name is empty" />;
  } else if (isNaN(parseInt(versionId))) {
    return <Message title={`Invalid Version ID ${versionId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelName} version ${versionId}` :
      `Unable to fetch model ${modelName} version ${versionId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!modelVersion) {
    return <Spinner tip={`Loading model ${modelName} version ${versionId} details...`} />;
  }

  return (
    <Page
      bodyNoPadding
      docTitle="Model Version Details"
      headerComponent={(
        <ModelVersionHeader
          modelVersion={modelVersion}
          onDeregisterVersion={deleteVersion}
          onSaveDescription={saveDescription}
          onSaveName={saveName}
          onUpdateTags={saveVersionTags}
        />
      )}
      id="modelDetails">
      <Tabs
        defaultActiveKey="overview"
        style={{ height: 'auto' }}
        tabBarStyle={{ backgroundColor: 'var(--theme-colors-monochrome-17)', paddingLeft: 24 }}
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
              disabled={modelVersion.model.archived}
              metadata={modelVersion.metadata}
              onSave={saveMetadata}
            />
          </div>
        </TabPane>
        <TabPane key="notes" tab="Notes">
          <div className={css.base}>
            <NotesCard
              disabled={modelVersion.model.archived}
              notes={modelVersion.notes ?? ''}
              onSave={saveNotes}
            />
          </div>
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default ModelVersionDetails;
