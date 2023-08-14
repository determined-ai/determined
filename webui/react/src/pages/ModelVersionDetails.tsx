import { Card } from 'antd';
import type { TabsProps } from 'antd';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import InfoBox from 'components/InfoBox';
import Breadcrumb from 'components/kit/Breadcrumb';
import Notes from 'components/kit/Notes';
import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';
import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import MetadataCard from 'components/Metadata/MetadataCard';
import Page, { BreadCrumbRoute } from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { getModelVersion, patchModelVersion } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ValueOf } from 'types';
import { Metadata, ModelVersion, Note } from 'types';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { isAborted, isNotFound } from 'utils/service';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize } from 'utils/workload';

import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';
import css from './ModelVersionDetails.module.scss';
import { WorkspaceDetailsTab } from './WorkspaceDetails';

const TabType = {
  Model: 'model',
  Notes: 'notes',
} as const;

type Params = {
  modelId: string;
  tab?: ValueOf<typeof TabType>;
  versionNum: string;
};

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Model;

const ModelVersionDetails: React.FC = () => {
  const [modelVersion, setModelVersion] = useState<ModelVersion>();
  const { modelId: modelID, versionNum: versionNUM, tab } = useParams<Params>();
  const workspace = Loadable.getOrElse(
    undefined,
    useObservable(
      workspaceStore.getWorkspace(
        modelVersion ? Loaded(modelVersion.model.workspaceId) : NotLoaded,
      ),
    ),
  );
  const [pageError, setPageError] = useState<Error>();
  const navigate = useNavigate();
  const location = useLocation();
  const [tabKey, setTabKey] = useState(tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY);

  const modelId = modelID ?? '0';
  const versionNum = versionNUM ?? '0';

  const basePath = paths.modelVersionDetails(modelId, versionNum);

  const { canModifyModelVersion, loading: rbacLoading } = usePermissions();

  const fetchModelVersion = useCallback(async () => {
    try {
      const versionData = await getModelVersion({
        modelName: modelId,
        versionNum: parseInt(versionNum),
      });
      /**
       * TODO: can this compare againt prev instead of modelVersion, so that
       * modelVersion can be remove from deps? would need to get modelVersion
       * out of deps in order to repoll on change fn
       */
      setModelVersion((prev) => (!_.isEqual(versionData, modelVersion) ? versionData : prev));
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [modelId, modelVersion, pageError, versionNum]);

  usePolling(fetchModelVersion);

  const handleTabChange = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  useEffect(() => workspaceStore.fetch(), []);

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_TAB_KEY);
  }, [location.pathname, tab]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      if (window.location.pathname.includes(basePath))
        navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [basePath, navigate, tab, tabKey]);

  const saveMetadata = useCallback(
    async (editedMetadata: Metadata) => {
      try {
        await patchModelVersion({
          body: { metadata: editedMetadata, modelName: modelId },
          modelName: modelId,
          versionNum: parseInt(versionNum),
        });
        await fetchModelVersion();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save metadata.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModelVersion, modelId, versionNum],
  );

  const saveNotes = useCallback(
    async (notes: Note) => {
      const editedNotes = notes.contents;
      try {
        await patchModelVersion({
          body: { modelName: modelId, notes: editedNotes },
          modelName: modelId,
          versionNum: parseInt(versionNum),
        });
        await fetchModelVersion();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to update notes.',
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModelVersion, modelId, versionNum],
  );

  const saveVersionTags = useCallback(
    async (newTags: string[]) => {
      try {
        await patchModelVersion({
          body: { labels: newTags, modelName: modelId },
          modelName: modelId,
          versionNum: parseInt(versionNum),
        });
        fetchModelVersion();
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to save tags.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [fetchModelVersion, modelId, versionNum],
  );

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
      .map((key) => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));
    const hasExperiment = !!modelVersion.checkpoint.experimentId;
    return [
      {
        content: hasExperiment ? (
          <Breadcrumb>
            <Breadcrumb.Item>
              <Link path={paths.experimentDetails(modelVersion.checkpoint.experimentId || '')}>
                Experiment {modelVersion.checkpoint.experimentId}
              </Link>
            </Breadcrumb.Item>
            {!!modelVersion.checkpoint.trialId && (
              <Breadcrumb.Item>
                <Link
                  path={paths.trialDetails(
                    modelVersion.checkpoint.trialId,
                    modelVersion.checkpoint.experimentId,
                  )}>
                  Trial {modelVersion.checkpoint.trialId}
                </Link>
              </Breadcrumb.Item>
            )}
            {!!modelVersion.checkpoint.totalBatches && (
              <Breadcrumb.Item>Batch {modelVersion.checkpoint.totalBatches}</Breadcrumb.Item>
            )}
          </Breadcrumb>
        ) : (
          <Breadcrumb>
            <Breadcrumb.Item>Task {modelVersion.checkpoint.taskId}</Breadcrumb.Item>
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
        content: resources.map((resource) => renderResource(resource.name, resource.size)),
        label: 'Code',
      },
    ];
  }, [modelVersion?.checkpoint]);

  const validationMetrics = useMemo(() => {
    if (!modelVersion?.checkpoint) return [];
    const metrics = Object.entries(modelVersion?.checkpoint?.validationMetrics?.avgMetrics || {});
    return metrics.map((metric) => ({
      content: metric[1],
      label: metric[0],
    }));
  }, [modelVersion?.checkpoint]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!modelVersion) {
      return [];
    }

    return [
      {
        children: (
          <div className={css.base}>
            <Card title="Model Checkpoint">
              <InfoBox rows={checkpointInfo} separator />
            </Card>
            <Card title="Validation Metrics">
              <InfoBox rows={validationMetrics} separator />
            </Card>
            <MetadataCard
              disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}
              metadata={modelVersion.metadata}
              onSave={saveMetadata}
            />
          </div>
        ),
        key: TabType.Model,
        label: 'Model',
      },
      {
        children: (
          <div className={css.base}>
            <Notes
              disabled={modelVersion.model.archived || !canModifyModelVersion({ modelVersion })}
              disableTitle
              notes={{ contents: modelVersion.notes ?? '', name: 'Notes' }}
              onError={handleError}
              onSave={saveNotes}
            />
          </div>
        ),
        key: TabType.Notes,
        label: 'Notes',
      },
    ];
  }, [
    checkpointInfo,
    modelVersion,
    canModifyModelVersion,
    saveMetadata,
    saveNotes,
    validationMetrics,
  ]);

  if (!modelId) {
    return <Message title="Model name is empty" />;
  } else if (isNaN(parseInt(versionNum))) {
    return <Message title={`Invalid Version ID ${versionNum}`} />;
  } else if (pageError && !isNotFound(pageError)) {
    const message = `Unable to fetch model ${modelId} version ${versionNum}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (pageError && isNotFound(pageError)) {
    return <PageNotFound />;
  } else if (!modelVersion || rbacLoading) {
    return <Spinner spinning tip={`Loading model ${modelId} version ${versionNum} details...`} />;
  }
  const isUncategorized = !workspace?.id || workspace.id === 1;
  const pageBreadcrumb: BreadCrumbRoute[] = [
    isUncategorized
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
      path: isUncategorized
        ? paths.modelList()
        : paths.workspaceDetails(workspace.id, WorkspaceDetailsTab.ModelRegistry),
    },
    {
      breadcrumbName: `${modelVersion.model.name} (${modelId})`,
      path: paths.modelDetails(modelVersion.model.id.toString()),
    },
    {
      breadcrumbName: `Version ${modelVersion.version}`,
      path: paths.modelVersionDetails(
        modelVersion.model.id.toString(),
        modelVersion.version.toString(),
      ),
    },
  ];

  return (
    <Page
      bodyNoPadding
      breadcrumb={pageBreadcrumb}
      docTitle="Model Version Details"
      headerComponent={
        <ModelVersionHeader
          fetchModelVersion={fetchModelVersion}
          modelVersion={modelVersion}
          onUpdateTags={saveVersionTags}
        />
      }
      id="modelDetails"
      notFound={pageError && isNotFound(pageError)}>
      {/* TODO: Clean up once we standardize page layouts */}
      <div style={{ padding: 16 }}>
        <Pivot activeKey={tabKey} items={tabItems} onChange={handleTabChange} />
      </div>
    </Page>
  );
};

export default ModelVersionDetails;
