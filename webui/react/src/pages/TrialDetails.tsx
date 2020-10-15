import { Button, Col, Form, Input, Modal, Row, Select, Space, Table, Tooltip } from 'antd';
import { SelectValue } from 'antd/lib/select';
import axios from 'axios';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import BadgeTag from 'components/BadgeTag';
import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Icon from 'components/Icon';
import Message from 'components/Message';
import MetricSelectFilter from 'components/MetricSelectFilter';
import Page from 'components/Page';
import Section from 'components/Section';
import SelectFilter, { ALL_VALUE } from 'components/SelectFilter';
import Spinner, { Indicator } from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import TrialActions, { Action as TrialAction } from 'pages/TrialDetails/TrialActions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { routeAll } from 'routes/utils';
import { forkExperiment } from 'services/api';
import { getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { ApiState } from 'services/types';
import {
  CheckpointDetail, ExperimentDetails, MetricName, MetricType, RawJson, Step, TrialDetails,
} from 'types';
import { clone, numericSorter } from 'utils/data';
import { humanReadableFloat } from 'utils/string';
import { extractMetricNames, extractMetricValue } from 'utils/trial';
import { trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

import css from './TrialDetails.module.scss';
import { columns as defaultColumns } from './TrialDetails.table';
import TrialChart from './TrialDetails/TrialChart';

const { Option } = Select;

interface Params {
  trialId: string;
}

enum TrialInfoFilter {
  HasCheckpoint = 'Has Checkpoint',
  HasValidation = 'Has Validation',
  HasCheckpointOrValidation = 'Has Checkpoint or Validation',
}

const getTrialLength = (config?: RawJson): [string, number] | undefined => {
  if (!config) return undefined;
  const entries = Object.entries(config?.searcher.max_length || {});
  return entries[0] as [string, number] || [ 'batches', 100 ];
};

const setTrialLength = (experimentConfig: RawJson, length: number): void => {
  const trialLength = getTrialLength(experimentConfig);
  if (trialLength) experimentConfig.searcher.max_length = { [trialLength[0]]: length } ;
};

const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: Record<string, string>,
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
  const { trialId: trialIdParam } = useParams<Params>();
  const trialId = parseInt(trialIdParam);
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilter = storage.getWithDefault(
    STORAGE_CHECKPOINT_VALIDATION_KEY,
    TrialInfoFilter.HasCheckpointOrValidation,
  );
  const [ pageSize, setPageSize ] = useState(initLimit);
  const [ hasCheckpointOrValidation, setHasCheckpointOrValidation ] = useState(initFilter);
  const [ contModalVisible, setContModalVisible ] = useState(false);
  const [ contFormVisible, setContFormVisible ] = useState(false);
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ contModalConfig, setContModalConfig ] = useState('Loading');
  const [ contMaxLength, setContMaxLength ] = useState<number>();
  const [ contDescription, setContDescription ] = useState<string>('Loading');
  const [ contError, setContError ] = useState<string>();
  const [ form ] = Form.useForm();
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ experiment, setExperiment ] = useState<ExperimentDetails>();
  const [ trialDetails, setTrialDetails ] = useState<ApiState<TrialDetails>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source: axios.CancelToken.source(),
  });

  const trial = trialDetails.data;
  const hparams = trial?.hparams;
  const experimentId = trial?.experimentId;
  const experimentConfig = experiment?.config;
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;
  const storageChartMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_CHART_METRICS_KEY}`;
  const storageTableMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_TABLE_METRICS_KEY}`;

  const metricNames = useMemo(() => extractMetricNames(trial?.steps), [ trial?.steps ]);

  const upgradedConfig = useMemo(() => {
    if (!experiment?.configRaw) return;
    const configClone = clone(experiment.configRaw);
    upgradeConfig(configClone);
    return configClone;
  }, [ experiment?.configRaw ]);

  const trialLength = useMemo(() => {
    return getTrialLength(upgradedConfig);
  }, [ upgradedConfig ]);

  const columns = useMemo(() => {
    const checkpointRenderer = (_: string, record: Step) => {
      if (record.checkpoint) {
        const checkpoint: CheckpointDetail = {
          ...record.checkpoint,
          batch: record.numBatches + record.priorBatchesProcessed,
          experimentId,
          trialId: record.id,
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
        return value ? <Tooltip title={value}>
          <span>{humanReadableFloat(value)}</span>
        </Tooltip> : undefined;
      };
      return metricCol;
    };

    const { metric, smallerIsBetter } = experimentConfig?.searcher || {};
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
        title: <BadgeTag
          label={metricName.name}
          tooltip={metricName.type}>{metricName.type.substr(0, 1).toUpperCase()}</BadgeTag>,
      });
    });

    return newColumns;
  }, [ experimentConfig, experimentId, metrics ]);

  const steps = useMemo(() => {
    const data = trial?.steps || [];
    return hasCheckpointOrValidation as string === ALL_VALUE ?
      data : data.filter(step => {
        if (hasCheckpointOrValidation === TrialInfoFilter.HasCheckpoint) {
          return !!step.checkpoint;
        } else if (hasCheckpointOrValidation === TrialInfoFilter.HasValidation) {
          return !!step.validation;
        } else if (hasCheckpointOrValidation === TrialInfoFilter.HasCheckpointOrValidation) {
          return !!step.checkpoint || !!step.validation;
        }
        return false;
      });
  }, [ hasCheckpointOrValidation, trial?.steps ]);

  const fetchTrialDetails = useCallback(async () => {
    try {
      const response = await getTrialDetails({
        cancelToken: trialDetails.source?.token,
        id: trialId,
      });
      setTrialDetails(prev => ({ ...prev, data: response, isLoading: false }));
    } catch (e) {
      if (!trialDetails.error && !axios.isCancel(e)) {
        setTrialDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ trialDetails.error, trialDetails.source, trialId ]);

  const handleActionClick = useCallback((action: TrialAction) => (): void => {
    switch (action) {
      case TrialAction.Continue:
        setContFormVisible(true);
        break;
    }
  }, []);

  const setFreshContinueConfig = useCallback(() => {
    if (!upgradedConfig || !hparams) return;
    // do not reset the config if the modal is open
    if (contModalVisible || contFormVisible) return;
    const config = clone(upgradedConfig);
    const newDescription = `Continuation of trial ${trialId}, experiment` +
      ` ${experimentId} (${config.description})`;
    setContDescription(newDescription);
    const maxLength = trialLength && trialLength[1];
    if (maxLength !== undefined) setContMaxLength(maxLength);

    config.description = newDescription;
    if (maxLength) setTrialLength(config, maxLength);
    const newConfig = trialContinueConfig(config, hparams, trialId);
    setContModalConfig(yaml.safeDump(newConfig));
  }, [
    contFormVisible,
    contModalVisible,
    upgradedConfig,
    experimentId,
    hparams,
    trialId,
    trialLength,
  ]);

  const handleContFormCancel = useCallback(() => {
    setContFormVisible(false);
    setFreshContinueConfig();
    form.resetFields();
  }, [ setFreshContinueConfig, form ]);

  const handleContModalCancel = useCallback(() => {
    setContModalVisible(false);
    setFreshContinueConfig();
  }, [ setFreshContinueConfig ]);

  const updateStatesFromForm = useCallback(() => {
    if (!hparams || !trialId) return;
    const formValues = form.getFieldsValue();
    try {
      const expConfig = yaml.safeLoad(contModalConfig) as RawJson;
      expConfig.description = formValues.description;
      setTrialLength(expConfig, parseInt(formValues.maxLength));
      const updateConfig = trialContinueConfig(expConfig, hparams, trialId);
      setContModalConfig(yaml.safeDump(updateConfig));
      return updateConfig;
    } catch (e) {
      handleError({
        error: e,
        message: 'Failed to parse experiment config',
        publicMessage: 'Please check the experiment config. \
If the problem persists please contact support.',
        publicSubject: 'Failed to parse experiment config',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [ contModalConfig, form, hparams, trialId ]);

  const handleFormCreate = useCallback(async () => {
    if (!experimentId) return;
    const updatedConfig = updateStatesFromForm();
    try {
      const newExperiementId = await forkExperiment({
        experimentConfig: JSON.stringify(updatedConfig),
        parentId: experimentId,
      });
      routeAll(`/det/experiments/${newExperiementId}`);
    } catch (e) {
      handleError({
        error: e,
        message: 'Failed to continue trial',
        publicMessage: 'Check the experiment config. \
If the problem persists please contact support.',
        publicSubject: 'Failed to continue trial',
        silent: false,
        type: ErrorType.Api,
      });
      setContError(e.response?.data?.message || e.message);
      setContModalVisible(true);
    } finally {
      setContFormVisible(false);
    }
  }, [ experimentId, updateStatesFromForm ]);

  const handleConfigChange = useCallback((config: string) => {
    setContModalConfig(config);
    setContError(undefined);
  }, []);

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const handleHasCheckpointOrValidationSelect = useCallback((value: SelectValue): void => {
    const filter = value as unknown as TrialInfoFilter;
    if (value as string !== ALL_VALUE && !Object.values(TrialInfoFilter).includes(filter)) return;
    setHasCheckpointOrValidation(filter);
    storage.set(STORAGE_CHECKPOINT_VALIDATION_KEY, filter);
  }, [ setHasCheckpointOrValidation, storage ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);
    if (storageTableMetricsKey) storage.set(storageTableMetricsKey, value);
  }, [ storage, storageTableMetricsKey ]);

  const handleEditContConfig = useCallback(() => {
    updateStatesFromForm();
    setContFormVisible(false);
    setContModalVisible(true);
  }, [ updateStatesFromForm ]);

  const handleTableChange = useCallback((tablePagination) => {
    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPageSize(tablePagination.pageSize);
  }, [ storage ]);

  usePolling(fetchTrialDetails);

  useEffect(() => {
    return () => trialDetails.source?.cancel();
  }, [ trialDetails.source ]);

  useEffect(() => {
    try {
      setFreshContinueConfig();
    } catch (e) {
      handleError({
        error: e,
        message: 'failed to load experiment config',
        type: ErrorType.ApiBadResponse,
      });
      setContModalConfig('failed to load experiment config');
    }
  }, [ setFreshContinueConfig ]);

  useEffect(() => {
    if (experimentId === undefined) return;

    const fetchExperimentDetails = async () => {
      try {
        const response = await getExperimentDetails({
          cancelToken: trialDetails.source?.token,
          id: experimentId,
        });
        setExperiment(response);

        // Default to selecting config search metric only.
        const searcherName = response.config?.searcher?.metric;
        const defaultMetric = metricNames.find(metricName => {
          return metricName.name === searcherName && metricName.type === MetricType.Validation;
        });
        const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
        const initMetrics = storage.getWithDefault(storageTableMetricsKey || '', defaultMetrics);
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
    };

    fetchExperimentDetails();
  }, [ experimentId, metricNames, trialDetails.source, storage, storageTableMetricsKey ]);

  if (isNaN(trialId)) return <Message title={`Invalid Trial ID ${trialIdParam}`} />;
  if (trialDetails.error !== undefined) {
    const message = isNotFound(trialDetails.error) ?
      `Unable to find Trial ${trialId}` :
      `Unable to fetch Trial ${trialId}`;
    return <Message message={trialDetails.error.message} title={message} />;
  }
  if (!trial || !experiment || !upgradedConfig) return <Spinner />;

  const options = (
    <Space size="middle">
      <SelectFilter
        dropdownMatchSelectWidth={300}
        label="Show"
        value={hasCheckpointOrValidation}
        onSelect={handleHasCheckpointOrValidationSelect}>
        <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
        {Object.values(TrialInfoFilter).map(key => <Option key={key} value={key}>{key}</Option>)}
      </SelectFilter>
      {metrics && <MetricSelectFilter
        metricNames={metricNames}
        multiple
        value={metrics}
        onChange={handleMetricChange} />}
    </Space>
  );

  return (
    <Page
      backPath={`/det/experiments/${experimentId}`}
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: '/det/experiments' },
        {
          breadcrumbName: `Experiment ${experimentId}`,
          path: `/det/experiments/${experimentId}`,
        },
        { breadcrumbName: `Trial ${trialId}`, path: `/det/trials/${trialId}` },
      ]}
      options={<TrialActions
        trial={trial}
        trials={experiment.trials}
        onClick={handleActionClick}
        onSettled={fetchTrialDetails} />}
      showDivider
      subTitle={<Badge state={trial?.state} type={BadgeType.State} />}
      title={`Trial ${trialId}`}>
      <Row className={css.topRow} gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <TrialInfoBox experiment={experiment} trial={trial} />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <TrialChart
            metricNames={metricNames}
            steps={trial?.steps}
            storageKey={storageChartMetricsKey}
            validationMetric={experimentConfig?.searcher.metric} />
        </Col>
        <Col span={24}>
          <Section options={options} title="Trial Information">
            <Table
              columns={columns}
              dataSource={steps}
              loading={{
                indicator: <Indicator />,
                spinning: trialDetails.isLoading,
              }}
              pagination={getPaginationConfig(steps.length, pageSize)}
              rowClassName={defaultRowClassName(false)}
              rowKey="id"
              scroll={{ x: 1000 }}
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange} />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && experimentConfig && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experimentConfig}
        show={showCheckpoint}
        title={`Checkpoint for Batch ${activeCheckpoint.batch}`}
        onHide={handleCheckpointDismiss} />}
      <CreateExperimentModal
        config={contModalConfig}
        error={contError}
        okText="Continue Trial"
        parentId={experiment.id}
        title={`Continue Trial ${trialId}`}
        visible={contModalVisible}
        onCancel={handleContModalCancel}
        onConfigChange={handleConfigChange}
        onVisibleChange={setContModalVisible} />
      <Modal
        footer={<>
          <Button onClick={handleEditContConfig}>Edit Full Config</Button>
          <Button type="primary" onClick={handleFormCreate}>Continue Trial</Button>
        </>}
        style={{ minWidth: '60rem' }}
        title={`Continue Trial ${trialId} of Experiment ${experimentId}`}
        visible={contFormVisible}
        onCancel={handleContFormCancel}
      >
        <Form
          form={form}
          initialValues={{ description: contDescription, maxLength: contMaxLength }}
          labelCol={{ span: 8 }}
          name="basic"
        >
          <Form.Item
            label={`Max ${trialLength && trialLength[0]}`}
            name="maxLength"
            rules={[ { message: 'Please set max length', required: true } ]}
          >
            <Input type="number" />
          </Form.Item>

          <Form.Item
            label="Experiment description"
            name="description"
            rules={[
              { message: 'Please set a description for the new experiment', required: true },
            ]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </Page>
  );
};

export default TrialDetailsComp;
