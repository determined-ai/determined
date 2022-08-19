import React, { useRef } from 'react';

import { InteractiveTableSettings } from 'components/InteractiveTable';
import LearningCurveChart from 'components/LearningCurveChart';
import Page from 'components/Page';
import Section from 'components/Section';
import useSettings from 'hooks/useSettings';
import TrialTable from 'pages/TrialsComparison/Table/TrialTable';
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import { metricToKey } from 'utils/metric';

import useHighlight from '../../hooks/useHighlight';

import useTrialActions from './Actions/useTrialActions';
import {
  useTrialCollections,
} from './Collections/useTrialCollections';
import useLearningCurveData from './Metrics/useLearningCurveData';
import useMetricView from './Metrics/useMetricView';
import { trialsTableSettingsConfig } from './Table/settings';
import { useFetchTrials } from './Trials/useFetchTrials';
import css from './TrialsComparison.module.scss';

interface Props {
  projectId: string;
}

const TrialsComparison: React.FC<Props> = ({ projectId }) => {

  const tableSettingsHook = useSettings<InteractiveTableSettings>(trialsTableSettingsConfig);
  const { settings: tableSettings, updateSettings } = tableSettingsHook;

  const C = useTrialCollections(projectId);

  const trials = useFetchTrials({
    filters: C.filters,
    limit: tableSettings.tableLimit,
    offset: tableSettings.tableOffset,
    sorter: C.sorter,
  });

  const M = useMetricView(trials.metrics);

  const A = useTrialActions({
    filters: C.filters,
    openCreateModal: C.openCreateModal,
    sorter: C.sorter,
  });

  // const [ pageSize, setPageSize ] = useState(MINIMUM_PAGE_SIZE);
  // const pageRef = useRef<HTMLElement>(null);

  const highlights = useHighlight((trial: V1AugmentedTrial): number => trial.trialId);

  const containerRef = useRef<HTMLElement>(null);

  const chartSeries = useLearningCurveData(trials.ids, trials.metrics, trials.maxBatch);

  const chartData = M.view?.metric
    && metricToKey(M.view.metric)
    && chartSeries?.metrics?.[metricToKey(M.view.metric)];

  return (
    <Page className={css.base} containerRef={containerRef}>
      <Section
        bodyBorder
        bodyScroll
        filters={M.controls}>
        <div className={css.container}>
          <div className={css.chart}>
            {M.view?.metric && chartData && (
              <LearningCurveChart
                data={chartData}
                focusedTrialId={highlights.id}
                selectedMetric={M.view.metric}
                selectedScale={M.view.scale}
                selectedTrialIds={A.selectedTrials}
                trialIds={trials.ids}
                xValues={chartSeries.batches}
                onTrialFocus={highlights.focus}
              />
            )}
          </div>
          {A.dispatcher}
          <TrialTable
            actionsInterface={A}
            collectionsInterface={C}
            containerRef={containerRef}
            highlights={highlights}
            tableSettingsHook={tableSettingsHook}
            trialsWithMetadata={trials}
            // handleTableChange={handleTableChange}
            // pageSize={pageSize}
          />
        </div>
      </Section>
      {A.modalContextHolder}
      {C.modalContextHolder}
    </Page>

  );
};

export default TrialsComparison;
