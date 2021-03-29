import { Button, Col, Row, Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import axios from 'axios';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';
import { useHistory } from 'react-router-dom';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal, { CreateExperimentType } from 'components/CreateExperimentModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import Message, { MessageType } from 'components/Message';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelectFilter from 'components/MetricSelectFilter';
import Page from 'components/Page';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SelectFilter, { ALL_VALUE } from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import TrialActions, { Action as TrialAction } from 'pages/TrialDetails/TrialActions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { paths, routeAll } from 'routes/utils';
import { createExperiment, getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import {
  CheckpointDetail, ExperimentBase, MetricName, MetricType, RawJson, Step, TrialDetails,
  TrialHyperParameters,
} from 'types';
import { clone, isEqual } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { hasCheckpoint, hasCheckpointStep, workloadsToSteps } from 'utils/step';
import { extractMetricNames, extractMetricValue } from 'utils/trial';
import { terminalRunStates, trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

import css from './TrialDetails.module.scss';
import { columns as defaultColumns } from './TrialDetails.table';
import TrialChart from './TrialDetails/TrialChart';

const { Option } = Select;

interface Params {
  experimentId?: string;
  trialId: string;
}

enum TrialInfoFilter {
  Checkpoint = 'Has Checkpoint',
  Validation = 'Has Validation',
  CheckpointOrValidation = 'Has Checkpoint or Validation',
}

const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: TrialHyperParameters,
  trialId: number,
): RawJson => {
  return {
    ...experimentConfig,
    hyperparameters: trialHParamsToExperimentHParams(trialHparams),
    searcher: {
      max_length: experimentConfig.searcher.max_length,
      metric: experimentConfig.searcher.metric,
      name: 'single',
      smaller_is_better: experimentConfig.searcher.smaller_is_better,
      source_trial_id: trialId,
    },
  };
};

const STORAGE_PATH = 'trial-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_CHECKPOINT_VALIDATION_KEY = 'checkpoint-validation';
const STORAGE_CHART_METRICS_KEY = 'metrics/chart';
const STORAGE_TABLE_METRICS_KEY = 'metrics/table';

const TrialDetailsComp: React.FC = () => {
  const { experimentId: experimentIdParam, trialId: trialIdParam } = useParams<Params>();
  const trialId = parseInt(trialIdParam);
  const history = useHistory();
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilter = storage.getWithDefault(
    STORAGE_CHECKPOINT_VALIDATION_KEY,
    TrialInfoFilter.CheckpointOrValidation,
  );
  const [ pageSize, setPageSize ] = useState(initLimit);
  const [ showFilter, setShowFilter ] = useState(initFilter);
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ defaultMetrics, setDefaultMetrics ] = useState<MetricName[]>([]);
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ canceler ] = useState(new AbortController());
  const [ source ] = useState(axios.CancelToken.source());
  const [ trialDetails, setTrialDetails ] = useState<ApiState<TrialDetails>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source,
  });
  const [ contModalConfig, setContModalConfig ] = useState<RawJson>();
  const [ contModalError, setContModalError ] = useState<string>();
  const [ isContModalVisible, setIsContModalVisible ] = useState(false);

  const trial = trialDetails.data;
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;
  const storageChartMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_CHART_METRICS_KEY}`;
  const storageTableMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_TABLE_METRICS_KEY}`;

  const hasFiltersApplied = useMemo(() => {
    const metricsApplied = !isEqual(metrics, defaultMetrics);
    const checkpointValidationFilterApplied = showFilter as string !== ALL_VALUE;
    return metricsApplied || checkpointValidationFilterApplied;
  }, [ showFilter, metrics, defaultMetrics ]);

  const metricNames = useMemo(() => extractMetricNames(
    trial?.workloads || [],
  ), [ trial?.workloads ]);

  const columns = useMemo(() => {
    const checkpointRenderer = (_: string, record: Step) => {
      if (trial && record.checkpoint && hasCheckpointStep(record)) {
        const checkpoint = {
          ...record.checkpoint,
          batch: record.checkpoint.numBatches + record.checkpoint.priorBatchesProcessed,
          experimentId: trial?.experimentId,
          trialId: trial?.id,
        };
        return (
          <Tooltip title="View Checkpoint">
            <Button
              aria-label="View Checkpoint"
              icon={<Icon name="checkpoint" />}
              onClick={e => handleCheckpointShow(e, checkpoint)} />
          </Tooltip>
        );
      }
      return null;
    };

    const metricRenderer = (metricName: MetricName) => {
      const metricCol = (_: string, record: Step) => {
        const value = extractMetricValue(record, metricName);
        return value != null ? <HumanReadableFloat num={value} /> : undefined;
      };
      return metricCol;
    };

    const { metric, smallerIsBetter } = experiment?.config?.searcher || {};
    const newColumns = [ ...defaultColumns ].map(column => {
      if (column.key === 'checkpoint') column.render = checkpointRenderer;
      return column;
    });

    metrics.forEach(metricName => {
      const stateIndex = newColumns.findIndex(column => column.key === 'state');
      newColumns.splice(stateIndex, 0, {
        defaultSortOrder: metric && metric === metricName.name ?
          (smallerIsBetter ? 'ascend' : 'descend') : undefined,
        render: metricRenderer(metricName),
        sorter: (a, b) => numericSorter(
          extractMetricValue(a, metricName),
          extractMetricValue(b, metricName),
        ),
        title: <MetricBadgeTag metric={metricName} />,
      });
    });

    return newColumns;
  }, [ experiment?.config, metrics, trial ]);

  const workloadSteps = useMemo(() => {
    const data = trial?.workloads || [];
    const workloadSteps = workloadsToSteps(data);
    return showFilter as string === ALL_VALUE ?
      workloadSteps : workloadSteps.filter(wlStep => {
        if (showFilter === TrialInfoFilter.Checkpoint) {
          return hasCheckpoint(wlStep);
        } else if (showFilter === TrialInfoFilter.Validation) {
          return !!wlStep.validation;
        } else if (showFilter === TrialInfoFilter.CheckpointOrValidation) {
          return !!wlStep.checkpoint || !!wlStep.validation;
        }
        return false;
      });
  }, [ showFilter, trial?.workloads ]);

  const fetchExperimentDetails = useCallback(async () => {
    if (!trial) return;

    try {
      const response = await getExperimentDetails(
        { id: trial.experimentId },
        { signal: canceler.signal },
      );
      setExperiment(response);

      // Experiment id does not exist in route, reroute to the one with it
      if (!experimentIdParam) {
        history.replace(paths.trialDetails(trial.id, trial.experimentId));
      }

      // Default to selecting config search metric only.
      const searcherName = response.config?.searcher?.metric;
      const defaultMetric = metricNames.find(metricName => {
        return metricName.name === searcherName && metricName.type === MetricType.Validation;
      });
      const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
      setDefaultMetrics(defaultMetrics);
      const initMetrics = storage.getWithDefault(storageTableMetricsKey || '', defaultMetrics);
      setDefaultMetrics(defaultMetrics);
      setMetrics(initMetrics);
    } catch (e) {
      if (axios.isCancel(e)) return;
      handleError({
        error: e,
        message: 'Failed to load experiment details.',
        publicMessage: 'Failed to load experiment details.',
        publicSubject: 'Unable to fetch Trial Experiment Detail',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [
    canceler,
    experimentIdParam,
    history,
    metricNames,
    storage,
    storageTableMetricsKey,
    trial,
  ]);

  const fetchTrialDetails = useCallback(async () => {
    try {
      const response = await getTrialDetails({ id: trialId }, { signal: canceler.signal });
      setTrialDetails(prev => ({ ...prev, data: response, isLoading: false }));
    } catch (e) {
      if (!trialDetails.error && !isAborted(e)) {
        setTrialDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ canceler, trialDetails.error, trialId ]);

  const { stopPolling } = usePolling(fetchTrialDetails);

  const showContModal = useCallback(() => {
    if (experiment?.configRaw && trial) {
      const rawConfig = trialContinueConfig(clone(experiment.configRaw), trial.hparams, trial.id);
      rawConfig.description = [
        `Continuation of trial ${trial.id},`,
        `experiment ${trial.experimentId} (${rawConfig.description})`,
      ].join(' ');
      upgradeConfig(rawConfig);
      setContModalConfig(rawConfig);
    }
    setIsContModalVisible(true);
  }, [ experiment?.configRaw, trial ]);

  const handleContModalCancel = useCallback(() => {
    setIsContModalVisible(false);
  }, []);

  const handleContModalSubmit = useCallback(async (newConfig: string) => {
    if (!trial) return;

    try {
      const { id: newExperimentId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: trial.experimentId,
      });
      setIsContModalVisible(false);
      routeAll(paths.experimentDetails(newExperimentId));
    } catch (e) {
      handleError({
        error: e,
        message: 'Failed to continue trial',
        publicMessage: [
          'Check the experiment config.',
          'If the problem persists please contact support.',
        ].join(' '),
        publicSubject: 'Failed to continue trial',
        silent: false,
        type: ErrorType.Api,
      });
      setContModalError(e.response?.data?.message || e.message);
    }
  }, [ trial ]);

  const handleActionClick = useCallback((action: TrialAction) => (): void => {
    switch (action) {
      case TrialAction.Continue:
        showContModal();
        break;
    }
  }, [ showContModal ]);

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const handleHasCheckpointOrValidationSelect = useCallback((value: SelectValue): void => {
    const filter = value as unknown as TrialInfoFilter;
    if (value as string !== ALL_VALUE && !Object.values(TrialInfoFilter).includes(filter)) return;
    setShowFilter(filter);
    storage.set(STORAGE_CHECKPOINT_VALIDATION_KEY, filter);
  }, [ setShowFilter, storage ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);
    if (storageTableMetricsKey) storage.set(storageTableMetricsKey, value);
  }, [ storage, storageTableMetricsKey ]);

  const handleTableChange = useCallback((tablePagination) => {
    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPageSize(tablePagination.pageSize);
  }, [ storage ]);

  useEffect(() => {
    fetchExperimentDetails();
  }, [ fetchExperimentDetails ]);

  useEffect(() => {
    if (trialDetails.data && terminalRunStates.has(trialDetails.data.state)) {
      stopPolling();
    }
  }, [ trialDetails.data, stopPolling ]);

  useEffect(() => {
    return () => {
      source.cancel();
      canceler.abort();
    };
  }, [ canceler, source ]);

  if (isNaN(trialId)) return <Message title={`Invalid Trial ID ${trialIdParam}`} />;
  if (trialDetails.error !== undefined) {
    const message = isNotFound(trialDetails.error) ?
      `Unable to find Trial ${trialId}` :
      `Unable to fetch Trial ${trialId}`;
    return <Message
      message={trialDetails.error.message}
      title={message}
      type={MessageType.Warning} />;
  }
  if (!trial || !experiment) return <Spinner />;

  const options = (
    <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
      <SelectFilter
        dropdownMatchSelectWidth={300}
        label="Show"
        value={showFilter}
        onSelect={handleHasCheckpointOrValidationSelect}>
        <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
        {Object.values(TrialInfoFilter).map(key => <Option key={key} value={key}>{key}</Option>)}
      </SelectFilter>
      {metrics && <MetricSelectFilter
        defaultMetricNames={defaultMetrics}
        metricNames={metricNames}
        multiple
        value={metrics}
        onChange={handleMetricChange} />}
    </ResponsiveFilters>
  );

  return (
    <Page
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: paths.experimentList() },
        {
          breadcrumbName: `Experiment ${experiment.id}`,
          path: paths.experimentDetails(experiment.id),
        },
        {
          breadcrumbName: `Trial ${trialId}`,
          path: paths.trialDetails(trialId, experiment.id),
        },
      ]}
      options={<TrialActions
        trial={trial}
        onClick={handleActionClick}
        onSettled={fetchTrialDetails} />}
      stickyHeader
      subTitle={<Badge state={trial?.state} type={BadgeType.State} />}
      title={`Trial ${trialId}`}>
      <Row className={css.topRow} gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <TrialInfoBox experiment={experiment} trial={trial} />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <TrialChart
            defaultMetricNames={defaultMetrics}
            metricNames={metricNames}
            storageKey={storageChartMetricsKey}
            validationMetric={experiment?.config?.searcher.metric}
            workloads={trial?.workloads} />
        </Col>
        <Col span={24}>
          <Section options={options} title="Trial Information">
            <ResponsiveTable<Step>
              columns={columns}
              dataSource={workloadSteps}
              loading={trialDetails.isLoading}
              pagination={getPaginationConfig(workloadSteps.length, pageSize)}
              rowClassName={defaultRowClassName({ clickable: false })}
              rowKey="batchNum"
              scroll={{ x: 1000 }}
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange} />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && experiment?.config && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment?.config}
        show={showCheckpoint}
        title={`Checkpoint for Batch ${activeCheckpoint.batch}`}
        onHide={handleCheckpointDismiss} />}
      <CreateExperimentModal
        config={contModalConfig}
        error={contModalError}
        title={`Continue Trial ${trialId}`}
        type={CreateExperimentType.ContinueTrial}
        visible={isContModalVisible}
        onCancel={handleContModalCancel}
        onOk={handleContModalSubmit}
      />
    </Page>
  );
};

export default TrialDetailsComp;
