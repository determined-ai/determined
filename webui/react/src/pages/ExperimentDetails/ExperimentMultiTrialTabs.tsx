import { Tabs } from 'antd';
import React, { useCallback } from 'react';
import { useParams } from 'react-router';

import DynamicTabs from 'components/DynamicTabs';
import NotesCard from 'components/NotesCard';
import ExperimentTrials from 'pages/ExperimentDetails/ExperimentTrials';
import { paths } from 'routes/utils';
import { patchExperiment } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { ExperimentBase } from 'types';
import handleError from 'utils/error';

const CodeViewer = React.lazy(() => import('./CodeViewer/CodeViewer'));

const { TabPane } = Tabs;

enum TabType {
  Configuration = 'configuration',
  Trials = 'trials',
  Visualization = 'visualization',
  Notes = 'notes',
}

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

  const basePath = paths.experimentDetails(experiment.id);

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
    <DynamicTabs basePath={basePath} className="no-padding">
      <TabPane key="visualization" tab="Visualization">
        <React.Suspense fallback={<Spinner tip="Loading experiment visualization..." />}>
          <ExperimentVisualization
            basePath={`${basePath}/${TabType.Visualization}`}
            experiment={experiment}
          />
        </React.Suspense>
      </TabPane>
      <TabPane key="trials" tab="Trials">
        <ExperimentTrials experiment={experiment} pageRef={pageRef} />
      </TabPane>
      <TabPane key="code" tab="Code">
        <React.Suspense fallback={<Spinner tip="Loading code viewer..." />}>
          <CodeViewer
            experimentId={experiment.id}
            runtimeConfig={experiment.configRaw}
            submittedConfig={experiment.originalConfig}
          />
        </React.Suspense>
      </TabPane>
      <TabPane key="notes" tab="Notes">
        <NotesCard
          notes={experiment.notes ?? ''}
          style={{ border: 0 }}
          onSave={handleNotesUpdate}
        />
      </TabPane>
    </DynamicTabs>
  );
};

export default ExperimentMultiTrialTabs;
