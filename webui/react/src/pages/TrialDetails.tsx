import { Breadcrumb, Button, Col, Form, Input, Modal, Row, Space } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import CreateExperimentModal from 'components/CreateExperimentModal';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import TrialActions, { Action as TrialAction } from 'pages/TrialDetails/TrialActions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { routeAll } from 'routes';
import { forkExperiment } from 'services/api';
import { getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { TrialDetailsParams } from 'services/types';
import { ExperimentDetails, RawJson, TrialDetails } from 'types';
import { clone } from 'utils/data';
import { extractMetricNames } from 'utils/trial';
import { trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

import css from './TrialDetails.module.scss';
import TrialChart from './TrialDetails/TrialChart';

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
  const [ trial, triggerTrialRequest ] =
    useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id: trialId });
  const [ contModalVisible, setContModalVisible ] = useState(false);
  const [ contModalConfig, setContModalConfig ] = useState('Loading');
  const [ contFormVisible, setContFormVisible ] = useState<boolean>(false);
  const [ contMaxLength, setContMaxLength ] = useState<number>();
  const [ contDescription, setContDescription ] = useState<string>('Loading');
  const [ contError, setContError ] = useState<string>();
  const [ experiment, setExperiment ] = useState<ExperimentDetails>();
  const [ form ] = Form.useForm();

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
  }, [ ]);

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

  useEffect(() => {
    if (experimentId === undefined) return;
    getExperimentDetails({ id:experimentId })
      .then(experiment => {
        setExperiment(experiment);
      });
  }, [ experimentId, trialId ]);

  const handleEditContConfig = useCallback(() => {
    updateStatesFromForm();
    setContFormVisible(false);
    setContModalVisible(true);
  }, [ updateStatesFromForm ]);

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
          <Section title="Trial Information">
            {/* Table */}
          </Section>
        </Col>
      </Row>
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
