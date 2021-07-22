import React, { useCallback, useState } from 'react';

import CreateExperimentModal, { CreateExperimentType } from 'components/CreateExperimentModal';
import { paths, routeToReactUrl } from 'routes/utils';
import { createExperiment } from 'services/api';
import { ExperimentBase, RawJson, TrialDetails, TrialHyperparameters } from 'types';
import { clone } from 'utils/data';
import { trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

interface ShowProps {
  experiment: ExperimentBase;
  trial?: TrialDetails;
  type: CreateExperimentType;
}

interface ModalState {
  config: RawJson;
  error?: string;
  experiment?: ExperimentBase;
  title: string;
  trial?: TrialDetails;
  type: CreateExperimentType;
  visible: boolean;
}

interface ModalHooks {
  createModal: () => JSX.Element | null;
  showModal: (props: ShowProps) => void;
}

const trialContinueConfig = (
  experimentConfig: RawJson,
  trialHparams: TrialHyperparameters,
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

const useCreateExperimentModal = (): ModalHooks => {
  const [ state, setState ] = useState<ModalState>({
    config: {},
    title: '',
    type: CreateExperimentType.Fork,
    visible: false,
  });
  const [ error, setError ] = useState<string>();

  const showModal = useCallback(({ experiment, trial, type }: ShowProps) => {
    const isContinueTrial = type === CreateExperimentType.ContinueTrial;

    let title = '';
    let config = clone(experiment.configRaw);

    if (isContinueTrial && trial) {
      title = `Continue Trial ${trial.id}`;
      config = trialContinueConfig(config, trial.hyperparameters, trial.id);
      config.description = `Continuation of trial ${trial.id}, experiment ${experiment.id}` +
        (config.description ? ` (${config.description})` : '');
    } else if (!isContinueTrial) {
      title = `Fork Experiment ${experiment.id}`;
      if (config.description) config.description = `Fork of ${config.description}`;
    }

    upgradeConfig(config);
    setState({ config, experiment, title, trial, type, visible: true });
  }, []);

  const handleCancel = useCallback(() => {
    setState(prev => ({ ...prev, visible: false }));
  }, []);

  const handleSubmit = useCallback(async (newConfig: string) => {
    const isContinueTrial = state.type === CreateExperimentType.ContinueTrial;
    if (!state.experiment || (isContinueTrial && !state.trial)) return;

    try {
      const { id: newExperimentId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: state.experiment.id,
      });
      setState(prev => ({ ...prev, visible: false }));
      routeToReactUrl(paths.experimentDetails(newExperimentId));
    } catch (e) {
      let errorMessage = 'Unable to continue trial with the provided config.';
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      } else if (e.json) {
        const errorJSON = await e.json();
        errorMessage = errorJSON.error?.error;
      }
      setError(errorMessage);
    }
  }, [ state ]);

  const createModal = useCallback(() => {
    if (!state) return null;
    return (
      <CreateExperimentModal
        config={state.config}
        error={error}
        title={state.title}
        type={state.type}
        visible={state.visible}
        onCancel={handleCancel}
        onOk={handleSubmit}
      />
    );
  }, [ error, handleCancel, handleSubmit, state ]);

  return { createModal, showModal };
};

export default useCreateExperimentModal;
