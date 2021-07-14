import React, { forwardRef, useCallback, useImperativeHandle, useState } from 'react';

import CreateExperimentModal, { CreateExperimentType } from 'components/CreateExperimentModal';
import handleError, { ErrorType } from 'ErrorHandler';
import { paths, routeToReactUrl } from 'routes/utils';
import { createExperiment } from 'services/api';
import { ExperimentBase, RawJson, TrialDetails, TrialHyperParameters } from 'types';
import { clone } from 'utils/data';
import { trialHParamsToExperimentHParams, upgradeConfig } from 'utils/types';

export interface ContinueTrialHandles {
  show: () => void;
}

interface Props {
  experiment: ExperimentBase;
  ref?: React.Ref<ContinueTrialHandles>;
  trial: TrialDetails;
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

const ContinueTrial: React.FC<Props> = forwardRef(function ContinueTrial(
  { experiment, trial }: Props,
  ref?: React.Ref<ContinueTrialHandles>,
) {
  const [ contModalConfig, setContModalConfig ] = useState<RawJson>();
  const [ contModalError, setContModalError ] = useState<string>();
  const [ isVisible, setIsVisible ] = useState(false);

  const show = useCallback(() => {
    const rawConfig = trialContinueConfig(clone(experiment.configRaw), trial.hparams, trial.id);
    let newDescription = `Continuation of trial ${trial.id}, experiment ${trial.experimentId}`;
    if (rawConfig.description) newDescription += ` (${rawConfig.description})`;
    rawConfig.description = newDescription;
    upgradeConfig(rawConfig);
    setContModalConfig(rawConfig);
    setIsVisible(true);
  }, [ experiment.configRaw, trial ]);

  useImperativeHandle(ref, () => ({ show }));

  const handleContModalCancel = useCallback(() => {
    setIsVisible(false);
  }, []);

  const handleContModalSubmit = useCallback(async (newConfig: string) => {
    try {
      const { id: newExperimentId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: trial.experimentId,
      });
      setIsVisible(false);
      routeToReactUrl(paths.experimentDetails(newExperimentId));
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

  return (
    <CreateExperimentModal
      config={contModalConfig}
      error={contModalError}
      title={`Continue Trial ${trial.id}`}
      type={CreateExperimentType.ContinueTrial}
      visible={isVisible}
      onCancel={handleContModalCancel}
      onOk={handleContModalSubmit}
    />
  );
});

export default ContinueTrial;
