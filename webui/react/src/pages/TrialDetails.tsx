import {
  Breadcrumb, Button, Col, Form, Input, Modal, Row, Select, Space, Table, Tooltip,
} from 'antd';
import { SelectValue } from 'antd/es/select';
import { ColumnType } from 'antd/es/table';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import { defaultRowClassName } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import TrialActions, { Action as TrialAction } from 'pages/TrialDetails/TrialActions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { routeAll } from 'routes';
import { forkExperiment } from 'services/api';
import { getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { TrialDetailsParams } from 'services/types';
import { CheckpointDetail, ExperimentDetails, RawJson, Step, TrialDetails } from 'types';
import { clone, metricNameSorter, numericSorter } from 'utils/data';
import { extractMetricNames, extractMetricValue } from 'utils/trial';
import { trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

import css from './TrialDetails.module.scss';
import { columns as defaultColumns } from './TrialDetails.table';
import TrialChart from './TrialDetails/TrialChart';

const { OptGroup, Option } = Select;

interface Params {
  trialId: string;
}

const getTrialLength = (config?: RawJson): [string, number] | undefined => {
  const entries = Object.entries(config?.searcher.max_length || {});
  if (entries.length !== 0) return entries[0] as [string, number];
  return undefined;
};

const setTrialLength = (experimentConfig: RawJson, length: number): void => {
  const trialLength = getTrialLength(experimentConfig);
  if (trialLength) experimentConfig.searcher.max_length = { [trialLength[0]]: length } ;
};

const trialContinueConfig =
(experimentConfig: RawJson, trialHparams: Record<string, string>, trialId: number): RawJson => {

  return {
    ...experimentConfig,
    hyperparameters: trialHParamsToExperimentHParams(trialHparams),
    searcher: {
      max_length: experimentConfig.searcher.max_length,
      metric: experimentConfig.searcher.metric,
      name: experimentConfig.searcher.name,
      smaller_is_better: experimentConfig.searcher.smaller_is_better,
      source_trial_id: trialId,
    },
  };
};

const TrialDetailsComp: React.FC = () => {
  const { trialId: trialIdParam } = useParams<Params>();
  const trialId = parseInt(trialIdParam);
  const [ experiment, setExperiment ] = useState<ExperimentDetails>();
  const [ contModalVisible, setContModalVisible ] = useState(false);
  const [ contModalConfig, setContModalConfig ] = useState('Loading');
  const [ contFormVisible, setContFormVisible ] = useState<boolean>(false);
  const [ contMaxLength, setContMaxLength ] = useState<number>();
  const [ contDescription, setContDescription ] = useState<string>('Loading');
  const [ contError, setContError ] = useState<string>();
  const [ form ] = Form.useForm();
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ metric, setMetric ] = useState<string[]>([]);
  const [ trial, triggerTrialRequest ] =
    useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id: trialId });

  const metricNames = useMemo(() => extractMetricNames(trial.data?.steps), [ trial.data?.steps ]);

  const upgradedConfig = useMemo(
    () => {
      if (!experiment?.configRaw) return;
      const configClone = clone(experiment.configRaw);
      upgradeConfig(configClone);
      return configClone;
    },
    [ experiment?.configRaw ],
  );

  const trialLength = useMemo(() => {
    return getTrialLength(upgradedConfig);
  }, [ upgradedConfig ]);

  const columns = useMemo(() => {
    const checkpointRenderer = (_: string, record: Step) => {
      if (record.checkpoint) {
        const checkpoint: CheckpointDetail = {
          ...record.checkpoint,
          batch: record.numBatches + record.priorBatchesProcessed,
          experimentId: trial.data?.experimentId,
          trialId: record.id,
        };
        return <Tooltip title="View Checkpoint">
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" />}
            onClick={e => handleCheckpointShow(e, checkpoint)} />
        </Tooltip>;
      }
      return null;
    };

    const newColumns: ColumnType<Step>[] = [ ...defaultColumns ];

    metric.forEach(metricName => {
      const stateIndex = newColumns.findIndex(column => /state/i.test(column.title as string));
      newColumns.splice(stateIndex, 0, {
        render: (_: string, record: Step) => extractMetricValue(record, metricName),
        sorter: (a, b) => numericSorter(
          extractMetricValue(a, metricName),
          extractMetricValue(b, metricName),
        ),
        title: metricName,
      });
    });

    newColumns.push({
      fixed: 'right',
      render: checkpointRenderer,
      title: 'Checkpoint',
      width: 100,
    });

    return newColumns;
  }, [ metric, trial.data?.experimentId ]);

  const pollTrialDetails = useCallback(
    () => triggerTrialRequest({ id: trialId }),
    [ triggerTrialRequest, trialId ],
  );
  usePolling(pollTrialDetails);
  const experimentId = trial.data?.experimentId;
  const hparams = trial.data?.hparams;

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

  const onConfigChange = useCallback( (config: string) => {
    setContModalConfig(config);
    setContError(undefined);
  }, []);

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

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const handleMetricSelect = useCallback((value: SelectValue) => {
    const metricName = value as string;
    setMetric(prev => {
      const newMetric = [ ...prev ];
      if (newMetric.indexOf(metricName) === -1) newMetric.push(metricName);
      return newMetric.sort(metricNameSorter(metricNames));
    });
  }, [ metricNames ]);

  const handleMetricDeselect = useCallback((value: SelectValue) => {
    if (metric.length <= 1) return;

    const metricName = value as string;
    setMetric(prev => {
      const newMetric = [ ...prev ];
      const index = newMetric.indexOf(metricName);
      if (index !== -1) newMetric.splice(index, 1);
      return newMetric.sort(metricNameSorter(metricNames));
    });
  }, [ metric, metricNames ]);

  useEffect(() => {
    if (experimentId === undefined) return;
    getExperimentDetails({ id:experimentId })
      .then(experiment => setExperiment(experiment));
  }, [ experimentId ]);

  const handleEditContConfig = useCallback(() => {
    updateStatesFromForm();
    setContFormVisible(false);
    setContModalVisible(true);
  }, [ updateStatesFromForm ]);

  /*
   * By default enable all metric columns for table because:
   * 1. The metric columns as sorted by order of relevance.
   * 2. The table supports horizontal scrolling to show additional columns.
   */
  useEffect(() => {
    if (metric && metric?.length !== 0) return;
    if (metricNames.training.length === 0 && metricNames.validation.length === 0) return;
    setMetric([
      ...metricNames.validation,
      ...metricNames.training,
    ].sort(metricNameSorter(metricNames)));
  }, [ metric, metricNames ]);

  if (isNaN(trialId)) {
    return (
      <Page id="page-error-message">
        <Message>Bad trial ID {trialIdParam}</Message>
      </Page>
    );
  }

  if (trial.error !== undefined) {
    const message = isNotFound(trial.error) ? `Trial ${trialId} not found.`
      : `Failed to fetch trial ${trialId}.`;
    return (
      <Page id="page-error-message">
        <Message>{message}</Message>
      </Page>
    );
  }

  if (!trial.data || !experiment || !upgradedConfig) {
    return <Spinner fillContainer />;
  }

  const options = metric ? (
    <SelectFilter
      enableSearchFilter={false}
      label="Metric"
      mode="multiple"
      showSearch={false}
      style={{ minWidth: 220 }}
      value={metric}
      onDeselect={handleMetricDeselect}
      onSelect={handleMetricSelect}>
      {metricNames.validation.length > 0 && <OptGroup label="Validation Metrics">
        {metricNames.validation.map(key => <Option key={key} value={key}>{key}</Option>)}
      </OptGroup>}
      {metricNames.training.length > 0 && <OptGroup label="Training Metrics">
        {metricNames.training.map(key => <Option key={key} value={key}>{key}</Option>)}
      </OptGroup>}
    </SelectFilter>
  ) : undefined;

  return (
    <Page title={`Trial ${trialId}`}>
      <Breadcrumb>
        <Breadcrumb.Item>
          <Space align="center" size="small">
            <Icon name="trial" size="small" />
            <Link path="/det/trials">Trials</Link>
          </Space>
        </Breadcrumb.Item>
        <Breadcrumb.Item>
          <span>{trialId}</span>
        </Breadcrumb.Item>
      </Breadcrumb>
      <TrialActions trial={trial.data}
        onClick={handleActionClick}
        onSettled={pollTrialDetails} />
      <Row className={css.topRow} gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <TrialInfoBox experiment={experiment} trial={trial.data} />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <TrialChart
            metricNames={metricNames}
            steps={trial.data?.steps}
            validationMetric={experiment.config?.searcher.metric} />
        </Col>
        <Col span={24}>
          <Section options={options} title="Trial Information">
            <Table
              columns={columns}
              dataSource={trial.data?.steps}
              loading={!trial.hasLoaded}
              rowClassName={defaultRowClassName()}
              rowKey="id"
              scroll={{ x: 1000 }}
              showSorterTooltip={false}
              size="small" />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
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
        onConfigChange={onConfigChange}
        onVisibleChange={setContModalVisible}
      />
      <Modal
        footer={<>
          <Button onClick={handleEditContConfig}>Edit Full Config</Button>
          <Button type="primary" onClick={handleFormCreate}>Continue Trial</Button>
        </>}
        style={{
          minWidth: '60rem',
        }}
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
