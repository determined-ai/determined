import { Tabs } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import NotesCard from 'components/NotesCard';
import Spinner from 'components/Spinner';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import { patchExperiment } from 'services/api';
import { ExperimentBase, ExperimentVisualizationType } from 'types';

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Trials = 'trials',
  Visualization = 'visualization',
  Notes = 'notes',
}

interface Params {
  tab?: TabType;
  viz?: ExperimentVisualizationType;
}

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Visualization;

const ExperimentConfiguration = React.lazy(() => {
  return import('./ExperimentConfiguration');
});
const ExperimentVisualization = React.lazy(() => {
  return import('./ExperimentVisualization');
});

export interface Props {
  experiment: ExperimentBase;
  fetchExperimentDetails: () => void;
}

const ExperimentMultiTrialTabs: React.FC<Props> = (
  { experiment, fetchExperimentDetails }: Props,
) => {
  const { tab, viz } = useParams<Params>();
  const history = useHistory();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [ tabKey, setTabKey ] = useState(defaultTabKey);

  const basePath = paths.experimentDetails(experiment.id);

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

  const handleNotesUpdate = useCallback(async (editedNotes: string) => {
    try {
      await patchExperiment({ body: { notes: editedNotes }, experimentId: experiment.id });
      await fetchExperimentDetails();
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to update experiment notes.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ experiment.id, fetchExperimentDetails ]);

  return (
    <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="visualization" tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <ExperimentVisualization
            basePath={`${basePath}/${TabType.Visualization}`}
            experiment={experiment}
            type={viz}
          />
        </React.Suspense>
      </TabPane>
      <TabPane key="trials" tab="Trials">
        <ExperimentTrials experiment={experiment} />
      </TabPane>
      <TabPane key="configuration" tab="Configuration">
        <React.Suspense fallback={<Spinner tip="Loading text editor..." />}>
          <ExperimentConfiguration experiment={experiment} />
        </React.Suspense>
      </TabPane>
      <TabPane key="notes" tab="Notes">
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
