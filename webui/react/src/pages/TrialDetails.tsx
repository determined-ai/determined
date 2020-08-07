import { Breadcrumb, Space } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
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
import { getExperimentDetails, getTrialDetails, isNotFound } from 'services/api';
import { TrialDetailsParams } from 'services/types';
import { ExperimentDetails, TrialDetails } from 'types';
import { clone } from 'utils/data';
import { trialHParamsToExperimentHParams } from 'utils/types';

interface Params {
  trialId: string;
}

const TrialDetailsComp: React.FC = () => {
  const { trialId: trialIdParam } = useParams<Params>();
  const trialId = parseInt(trialIdParam);
  const [ trial, triggerTrialRequest ] =
    useRestApi<TrialDetailsParams, TrialDetails>(getTrialDetails, { id: trialId });
  const [ contModalVisible, setContModalVisible ] = useState(false);
  const [ contModalConfig, setContModalConfig ] = useState('Loading');

  const pollTrialDetails = useCallback(
    () => triggerTrialRequest({ id: trialId }),
    [ triggerTrialRequest, trialId ],
  );
  usePolling(pollTrialDetails);
  const experimentId = trial.data?.experimentId;
  const hparams = trial.data?.hparams;

  useEffect(() => {
    if (!experimentId || !hparams) return;
    getExperimentDetails({ id: experimentId })
      .then(experiment => {

        try {
          const rawConfig = clone(experiment.configRaw);
          const newDescription = `Continuation of trial ${trialId}, experiment` +
          ` ${experiment.id} (${rawConfig.description})`;
          const newSearcher = {
            max_steps: 100, // TODO add form
            metric: rawConfig.searcher.metric,
            name: rawConfig.searcher.name,
            smaller_is_better: rawConfig.searcher.smaller_is_better,
            source_trial_id: trialId,
          };
          const newHyperparameters = trialHParamsToExperimentHParams(hparams);

          setContModalConfig(yaml.safeDump({
            ...rawConfig,
            description: newDescription,
            hyperparameters: newHyperparameters,
            searcher: newSearcher,
          }));
        } catch (e) {
          setContModalConfig('failed to load experiment config');
          handleError({
            error: e,
            message: 'failed to load experiment config',
            type: ErrorType.ApiBadResponse,
          });
        }
      });

  } , [ experimentId, trialId, hparams ]);

  const handleActionClick = useCallback((action: TrialAction) => (): void => {
    switch (action) {
      case TrialAction.Continue:
        setContModalVisible(true);
        break;
    }
  }, [ ]);

  const [ experiment, setExperiment ] = useState<ExperimentDetails>();

  useEffect(() => {
    // TODO find a solution to conditional polling
    if (experimentId === undefined) return;
    getExperimentDetails({ id:experimentId })
      .then(experiment => setExperiment(experiment));
  }, [ experimentId ]);

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

  if (!trial.data || !experiment) {
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
      <Section title="Info Box">
        <TrialInfoBox experiment={experiment} trial={trial.data} />
      </Section>
      <Section title="Chart" />
      <Section title="Steps" />
      <CreateExperimentModal
        config={contModalConfig}
        okText="Create"
        parentId={experiment.id}
        title={`Continue Trial ${trialId}`}
        visible={contModalVisible}
        onConfigChange={setContModalConfig}
        onVisibleChange={setContModalVisible}
      />
    </Page>
  );
};

export default TrialDetailsComp;
