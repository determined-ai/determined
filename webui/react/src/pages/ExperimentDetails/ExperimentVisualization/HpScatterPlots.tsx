import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import BadgeTag from 'components/BadgeTag';
import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScatterPlot from 'components/ScatterPlot';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentBase, ExperimentHyperParamType, MetricName, metricTypeParamMap } from 'types';
import { isNumber } from 'utils/data';
import { terminalRunStates } from 'utils/types';

import css from './HpScatterPlots.module.scss';

const { Option } = Select;

interface Props {
  batches: number[];
  experiment: ExperimentBase;
  hParams: string[];
  metrics: MetricName[];
  onBatchChange?: (batch: number) => void;
  onHParamChange?: (hParams?: string[]) => void;
  onMetricChange?: (metric: MetricName) => void;
  selectedBatch: number;
  selectedHParams: string[];
  selectedMetric: MetricName;
}

interface HpMetricData {
  hpLogScales: Record<string, boolean>;
  hpValues: Record<string, number[]>;
  metricValues: Record<string, number[]>;
  trialIds: number[];
}

const ScatterPlots: React.FC<Props> = ({
  batches,
  experiment,
  hParams,
  metrics,
  onBatchChange,
  onHParamChange,
  onMetricChange,
  selectedBatch,
  selectedHParams,
  selectedMetric,
}: Props) => {
  const baseRef = useRef<HTMLDivElement>(null);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<HpMetricData>();
  const [ pageError, setPageError ] = useState<Error>();
  const resize = useResize(baseRef);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const resetData = useCallback(() => {
    setChartData(undefined);
    setHasLoaded(false);
  }, []);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    if (!onBatchChange) return;
    resetData();
    onBatchChange(batch as number);
  }, [ onBatchChange, resetData ]);

  const handleHParamChange = useCallback((hps: SelectValue) => {
    if (!onHParamChange) return;
    if (Array.isArray(hps)) {
      onHParamChange(hps.length === 0 ? undefined : hps as string[]);
    }
  }, [ onHParamChange ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (!onMetricChange) return;
    resetData();
    onMetricChange(metric);
  }, [ onMetricChange, resetData ]);

  useEffect(() => {
    const canceler = new AbortController();

    const trialIds: number[] = [];
    const hpTrialMap: Record<string, Record<number, { hp: number, metric: number }>> = {};

    consumeStream<V1TrialsSnapshotResponse>(
      detApi.StreamingInternal.determinedTrialsSnapshot(
        experiment.id,
        selectedBatch,
        selectedMetric.name,
        metricTypeParamMap[selectedMetric.type],
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event || !event.trials || !Array.isArray(event.trials)) return;

        const hpMetricMap: Record<string, number[]> = {};
        const hpValueMap: Record<string, number[]> = {};
        const hpLogScaleMap: Record<string, boolean> = {};

        event.trials.forEach(trial => {
          const trialId = trial.trialId;
          trialIds.push(trialId);

          hParams.forEach(hParam => {
            if (!isNumber(trial.hparams[hParam])) return;

            hpTrialMap[hParam] = hpTrialMap[hParam] || {};
            hpTrialMap[hParam][trialId] = hpTrialMap[hParam][trialId] || {};
            hpTrialMap[hParam][trialId] = {
              hp: trial.hparams[hParam],
              metric: trial.metric,
            };
          });
        });

        hParams.forEach(hParam => {
          const hp = (experiment.config.hyperparameters || {})[hParam];
          if (hp.type === ExperimentHyperParamType.Log) hpLogScaleMap[hParam] = true;

          hpMetricMap[hParam] = [];
          hpValueMap[hParam] = [];
          trialIds.forEach(trialId => {
            hpMetricMap[hParam].push(hpTrialMap[hParam][trialId].metric);
            hpValueMap[hParam].push(hpTrialMap[hParam][trialId].hp);
          });
        });

        setChartData({
          hpLogScales: hpLogScaleMap,
          hpValues: hpValueMap,
          metricValues: hpMetricMap,
          trialIds,
        });
        setHasLoaded(true);
      },
    ).catch(e => setPageError(e));

    return () => canceler.abort();
  }, [ experiment, hParams, selectedBatch, selectedMetric ]);

  if (pageError) {
    return <Message title={pageError.message} />;
  } else if (hasLoaded && !chartData) {
    return isExperimentTerminal ? (
      <Message title="No data to plot." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot." />
        <Spinner />
      </div>
    );
  }

  let content = <Spinner />;
  if (hasLoaded && chartData) {
    if (chartData.trialIds.length === 0) {
      content = <Message title="No data to plot." type={MessageType.Empty} />;
    } else {
      content = (
        <>
          <div className={css.legend}>
            <div className={css.legendItem}>
              <b>x-axis</b> = hyperparameters
            </div>
            <div className={css.legendItem}>
              <b>y-axis</b> =&nbsp;
              <BadgeTag
                label={selectedMetric.name}
                tooltip={selectedMetric.type}>{selectedMetric.type.substr(0, 1).toUpperCase()}
              </BadgeTag>
            </div>
          </div>
          <Grid
            border={true}
            minItemWidth={resize.width > 320 ? 35 : 27}
            mode={GridMode.AutoFill}>
            {selectedHParams.map(hParam => (
              <ScatterPlot
                key={hParam}
                title={hParam}
                x={chartData.hpValues[hParam]}
                xLogScale={chartData.hpLogScales[hParam]}
                y={chartData.metricValues[hParam]} />
            ))}
          </Grid>
        </>
      );
    }
  }

  return (
    <div className={css.base} ref={baseRef}>
      <Section
        options={<ResponsiveFilters>
          <SelectFilter
            enableSearchFilter={false}
            label="Batches Processed"
            showSearch={false}
            value={selectedBatch}
            onChange={handleBatchChange}>
            {batches.map(batch => <Option key={batch} value={batch}>{batch}</Option>)}
          </SelectFilter>
          <MetricSelectFilter
            defaultMetricNames={metrics}
            label="Metric"
            metricNames={metrics}
            multiple={false}
            value={selectedMetric}
            width={'100%'}
            onChange={handleMetricChange} />
          <MultiSelect
            label="HP"
            value={selectedHParams}
            onChange={handleHParamChange}>
            {hParams.map(hpKey => <Option key={hpKey} value={hpKey}>{hpKey}</Option>)}
          </MultiSelect>
        </ResponsiveFilters>}
        title="HP Scatter Plots">
        <div className={css.container}>{content}</div>
      </Section>
    </div>
  );
};

export default ScatterPlots;
