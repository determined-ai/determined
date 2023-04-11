import type { TabsProps } from 'antd';
import { string } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import NotesCard from 'components/NotesCard';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import ExperimentCodeViewer from 'pages/ExperimentDetails/ExperimentCodeViewer';
import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import { patchExperiment } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { ValueOf } from 'shared/types';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { ExperimentBase } from 'types';
import handleError from 'utils/error';

import { ExperimentVisualizationType } from './ExperimentVisualization';

const TabType = {
  Code: 'code',
  Notes: 'notes',
  Trials: 'trials',
  Visualization: 'visualization',
} as const;

type Params = {
  tab?: ValueOf<typeof TabType>;
  viz?: ExperimentVisualizationType;
};

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Visualization;

const ExperimentVisualization = React.lazy(() => {
  return import('./ExperimentVisualization');
});

export interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
  pageRef: React.RefObject<HTMLElement>;
}

const ExperimentMultiTrialTabs: React.FC<Props> = ({
  experiment,
  fetchExperimentDetails,
  pageRef,
}: Props) => {
  const { tab, viz } = useParams<Params>();
  const navigate = useNavigate();
  const location = useLocation();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [tabKey, setTabKey] = useState(defaultTabKey);

  const basePath = paths.experimentDetails(experiment.id);

  const configForExperiment = (experimentId: number): SettingsConfig<{ filePath: string }> => ({
    settings: {
      filePath: {
        defaultValue: '',
        storageKey: 'filePath',
        type: string,
      },
    },
    storagePath: `selected-file-${experimentId}`,
  });
  const { settings, updateSettings } = useSettings<{ filePath: string }>(
    configForExperiment(experiment.id),
  );
  const handleSelectFile = useCallback(
    (filePath: string) => {
      updateSettings({ filePath });
    },
    [updateSettings],
  );

  const handleTabChange = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

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

  const handleNotesUpdate = useCallback(
    async (editedNotes: string) => {
      try {
        await patchExperiment({ body: { notes: editedNotes }, experimentId: experiment.id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to update experiment notes.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [experiment.id, fetchExperimentDetails],
  );

  const { canModifyExperimentMetadata, canViewExperimentArtifacts } = usePermissions();
  const showExperimentArtifacts = canViewExperimentArtifacts({
    workspace: { id: experiment.workspaceId },
  });
  const editableNotes = canModifyExperimentMetadata({
    workspace: { id: experiment.workspaceId },
  });

  const tabItems: TabsProps['items'] = useMemo(() => {
    const items: TabsProps['items'] = [
      {
        children: (
          <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
            <ExperimentVisualization
              basePath={`${basePath}/${TabType.Visualization}`}
              experiment={experiment}
              type={viz}
            />
          </React.Suspense>
        ),
        key: TabType.Visualization,
        label: 'Visualization',
      },
      {
        children: <ExperimentTrials experiment={experiment} pageRef={pageRef} />,
        key: TabType.Trials,
        label: 'Trials',
      },
    ];

    if (showExperimentArtifacts) {
      items.push({
        children: (
          <ExperimentCodeViewer
            experiment={experiment}
            selectedFilePath={settings.filePath}
            onSelectFile={handleSelectFile}
          />
        ),
        key: TabType.Code,
        label: 'Code',
      });
    }

    items.push({
      children: (
        <NotesCard
          disabled={!editableNotes}
          notes={experiment.notes ?? ''}
          style={{ border: 0 }}
          onSave={handleNotesUpdate}
        />
      ),
      key: TabType.Notes,
      label: 'Notes',
    });
    return items;
  }, [
    basePath,
    editableNotes,
    experiment,
    handleNotesUpdate,
    handleSelectFile,
    pageRef,
    settings.filePath,
    showExperimentArtifacts,
    viz,
  ]);

  return <Pivot activeKey={tabKey} items={tabItems} onChange={handleTabChange} />;
};

export default ExperimentMultiTrialTabs;
