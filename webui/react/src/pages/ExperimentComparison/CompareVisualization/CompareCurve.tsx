import { Alert } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Section from 'components/Section';
import TableBatch from 'components/TableBatch';
import { useStore } from 'contexts/Store';
import { openOrCreateTensorBoard } from 'services/api';
import { V1ExpTrial, V1TrialsSampleResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import { Scale } from 'types';
import {
  ExperimentAction as Action, CommandTask, ExperimentBase, Hyperparameter, HyperparameterType,
  MetricName,
  metricTypeParamMap,
} from 'types';
import handleError from 'utils/error';
import { openCommand } from 'wait';

import { ErrorLevel, ErrorType } from '../../../shared/utils/error';

import css from './CompareCurve.module.scss';
import HpTrialTable, { TrialHParams } from './CompareTable';

interface Props {
  filters?: React.ReactNode;
  // fullHParams: string[];
  selectedMaxTrial: number;
  selectedMetric: MetricName
  selectedScale: Scale;
  trialHps: TrialHParams[];
  chartData: (number | null)[][];
  trialIds: number[];
  hyperparameters: Record<string, Hyperparameter>;
  batches: number[]
  hasLoaded: boolean;

}
enum PageError {
  MetricBatches,
  MetricHpImportance,
  MetricNames,
  ExperimentSample
}
const PAGE_ERROR_MESSAGES = {
  [PageError.MetricBatches]: 'Unable to retrieve experiment batches info.',
  [PageError.MetricHpImportance]: 'Unable to retrieve experiment hp importance.',
  [PageError.MetricNames]: 'Unable to retrieve experiment metric info.',
  [PageError.ExperimentSample]: 'Unable to retrieve experiment info.',
};

const MAX_DATAPOINTS = 5000;

const LearningCurve: React.FC<Props> = ({

  filters,
  // fullHParams,
  selectedMaxTrial,
  selectedMetric,
  selectedScale,
  trialHps,
  chartData,
  trialIds,
  batches,
  hyperparameters,
  hasLoaded,
}: Props) => {
  const { ui } = useStore();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<number[]>([]);
  const [ highlightedTrialId, setHighlightedTrialId ] = useState<number>();

  const hasTrials = trialIds.length !== 0;
  // const isExperimentTerminal = terminalRunStates.has(experiment.state as RunState);

  const handleTrialFocus = useCallback((trialId: number | null) => {
    setHighlightedTrialId(trialId != null ? trialId : undefined);
  }, []);

  const handleTableMouseEnter = useCallback((event: React.MouseEvent, record: TrialHParams) => {
    if (record.id) setHighlightedTrialId(record.id);
  }, []);

  const handleTableMouseLeave = useCallback(() => {
    setHighlightedTrialId(undefined);
  }, []);

  const clearSelected = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  const sendBatchActions = useCallback(async (action: Action) => {
    if (action === Action.OpenTensorBoard) {
      return await openOrCreateTensorBoard({ trialIds: selectedRowKeys });
    }
  }, [ selectedRowKeys ]);

  const submitBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard && result) {
        openCommand(result as CommandTask);
      }
    } catch (e) {
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to View TensorBoard for Selected Trials' :
        `Unable to ${action} Selected Trials`;
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ sendBatchActions ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  if (hasLoaded && !hasTrials) {
    return (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner />
      </div>
    );
  }

  return (
    <div className={css.base}>
      <Section bodyBorder bodyScroll filters={filters} loading={!hasLoaded}>
        <div className={css.container}>
          <div className={css.chart}>
            <LearningCurveChart
              data={chartData}
              focusedTrialId={highlightedTrialId}
              selectedMetric={selectedMetric}
              selectedScale={selectedScale}
              selectedTrialIds={selectedRowKeys}
              trialIds={trialIds}
              xValues={batches}
              onTrialFocus={handleTrialFocus}
            />
          </div>
          <TableBatch
            actions={[
              { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
            ]}
            selectedRowCount={selectedRowKeys.length}
            onAction={action => submitBatchAction(action as Action)}
            onClear={clearSelected}
          />
          <HpTrialTable
            handleTableRowSelect={handleTableRowSelect}
            highlightedTrialId={highlightedTrialId}
            hyperparameters={hyperparameters}
            metric={selectedMetric}
            selectedRowKeys={selectedRowKeys}
            selection={true}
            trialHps={trialHps}
            trialIds={trialIds}
            onMouseEnter={handleTableMouseEnter}
            onMouseLeave={handleTableMouseLeave}
          />
        </div>
      </Section>

    </div>
  );
};

export default LearningCurve;
