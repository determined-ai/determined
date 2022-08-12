import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import NotesCard from 'components/NotesCard';
import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import { patchExperiment } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { ExperimentBase, ExperimentVisualizationType } from 'types';
import handleError from 'utils/error';

const CodeViewer = React.lazy(() => import('./CodeViewer/CodeViewer'));

const { TabPane } = Tabs;

enum TabType {
  Code = 'code',
  Trials = 'trials',
  Visualization = 'visualization',
  Notes = 'notes',
}

type Params = {
  tab?: TabType;
  viz?: ExperimentVisualizationType;
}

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

const ExperimentMultiTrialTabs: React.FC<Props> = (
  { experiment, fetchExperimentDetails, pageRef }: Props,
) => {
  const { tab, viz } = useParams<Params>();
  const navigate = useNavigate();
  const location = useLocation();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const basePath = paths.experimentDetails(experiment.id);

  const handleTabChange = useCallback((key) => {
    navigate(`${basePath}/${key}`, { replace: true });
  }, [ basePath, navigate ]);

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_TAB_KEY);
  }, [ location.pathname, tab ]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [ basePath, navigate, tab, tabKey ]);

  const handleNotesUpdate = useCallback(async (editedNotes: string) => {
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
  }, [ experiment.id, fetchExperimentDetails ]);

  return (
    <Tabs activeKey={tabKey} className="no-padding" onChange={handleTabChange}>
      <TabPane key={TabType.Visualization} tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <ExperimentVisualization
            basePath={`${basePath}/${TabType.Visualization}`}
            experiment={experiment}
            type={viz}
          />
        </React.Suspense>
      </TabPane>
      <TabPane key={TabType.Trials} tab="Trials">
        <ExperimentTrials experiment={experiment} pageRef={pageRef} />
      </TabPane>
      <TabPane key={TabType.Code} tab="Code">
        <React.Suspense fallback={<Spinner tip="Loading code viewer..." />}>
          <CodeViewer
            experimentId={experiment.id}
            runtimeConfig={experiment.configRaw}
            submittedConfig={experiment.originalConfig}
          />
        </React.Suspense>
      </TabPane>
      <TabPane key={TabType.Notes} tab="Notes">
        <NotesCard
          notes={experiment.notes ?? ''}
          style={{ border: 0 }}
          onSave={handleNotesUpdate}
        />
      </TabPane>
    </Tabs>
  );
};

export default ExperimentMultiTrialTabs;
