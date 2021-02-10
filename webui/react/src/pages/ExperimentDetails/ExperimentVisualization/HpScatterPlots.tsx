import { Alert, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message, { MessageType } from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ScatterPlot from 'components/ScatterPlot';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { V1TrialsSnapshotResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ShirtSize } from 'themes';
import { ExperimentBase, MetricName, metricTypeParamMap } from 'types';
import { hasObjectKeys, isObject } from 'utils/data';
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
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ chartData, setChartData ] = useState<number[]>();
  const [ pageError, setPageError ] = useState<Error>();
  const fullHpList = Object.keys(experiment.config.hyperparameters) || [];

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const x = useMemo(() => ([ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 ]), []);
  const y = useMemo(() => new Array(10).fill(null).map(() => Math.random()), []);

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

        console.log('event', event);
        setHasLoaded(true);
      },
    ).catch(e => setPageError(e));

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedMetric ]);

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

  return (
    <div className={css.base}>
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
            value={hParams}
            onChange={handleHParamChange}>
            {fullHpList.map(hpKey => <Option key={hpKey} value={hpKey}>{hpKey}</Option>)}
          </MultiSelect>
        </ResponsiveFilters>}
        title="Scatter Plots">
        <Grid gap={ShirtSize.big} mode={GridMode.AutoFill}>
          {hParams.map(hParam => (
            // <ScatterPlot data={} key={hParam} />
            hParam
          ))}
          {/* {new Array(100).fill(null).map((_, index) => (
            <ScatterPlot key={index} data={{ x, y, values: y }} />
          ))} */}
        </Grid>
      </Section>
    </div>
  );
};

export default ScatterPlots;
