import React, { useCallback, useMemo, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import { terminalRunStates } from 'constants/states';
import useCreateExperimentModal, {
  CreateExperimentType,
} from 'hooks/useModal/Experiment/useModalExperimentCreate';
import TrialHeaderLeft from 'pages/TrialDetails/Header/TrialHeaderLeft';
import { openOrCreateTensorBoard } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ExperimentAction as Action, ExperimentAction, ExperimentBase, TrialDetails } from 'types';
import { canUserActionExperiment } from 'utils/experiment';
import { isMetricsWorkload } from 'utils/workload';
import { openCommand } from 'wait';

export const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const workloadsWithSomeMetric = trial.workloads
    .map(w => w.checkpoint || w.training || w.validation)
    .filter(workload => workload && isMetricsWorkload(workload) && !!workload.metrics);
  return isTerminal && workloadsWithSomeMetric.length === 0;
};

interface Props {
  experiment: ExperimentBase;
  fetchTrialDetails: () => void;
  trial: TrialDetails;
}

const TrialDetailsHeader: React.FC<Props> = ({
  experiment,
  fetchTrialDetails,
  trial,
}: Props) => {
  const [ isRunningTensorBoard, setIsRunningTensorBoard ] = useState<boolean>(false);

  const handleModalClose = useCallback(() => fetchTrialDetails(), [ fetchTrialDetails ]);

  const { contextHolder, modalOpen } = useCreateExperimentModal({ onClose: handleModalClose });

  const handleContinueTrial = useCallback(() => {
    modalOpen({ experiment, trial, type: CreateExperimentType.ContinueTrial });
  }, [ experiment, modalOpen, trial ]);

  const headerOptions = useMemo<Option[]>(() => {
    const options: Option[] = [];

    if (!trialWillNeverHaveData(trial)) {
      options.push({
        icon: <Icon name="tensor-board" size="small" />,
        isLoading: isRunningTensorBoard,
        key: Action.OpenTensorBoard,
        label: 'TensorBoard',
        onClick: async () => {
          setIsRunningTensorBoard(true);
          const tensorboard = await openOrCreateTensorBoard({ trialIds: [ trial.id ] });
          openCommand(tensorboard);
          await fetchTrialDetails();
          setIsRunningTensorBoard(false);
        },
      });
    }

    if (canUserActionExperiment(undefined, ExperimentAction.ContinueTrial, experiment, trial)) {
      if (trial.bestAvailableCheckpoint !== undefined) {
        options.push({
          icon: <Icon name="fork" size="small" />,
          key: Action.ContinueTrial,
          label: 'Continue Trial',
          onClick: handleContinueTrial,
        });
      } else {
        options.push({
          icon: <Icon name="fork" size="small" />,
          key: Action.ContinueTrial,
          label: 'Continue Trial',
          tooltip: 'No checkpoints found. Cannot continue trial',
        });
      }
    }

    return options;
  }, [
    experiment,
    fetchTrialDetails,
    handleContinueTrial,
    isRunningTensorBoard,
    trial,
  ]);

  return (
    <>
      <BreadcrumbBar experiment={experiment} id={trial.id} trial={trial} type="trial" />
      <PageHeaderFoldable
        leftContent={<TrialHeaderLeft experiment={experiment} trial={trial} />}
        options={headerOptions}
      />
      {contextHolder}
    </>
  );
};

export default TrialDetailsHeader;
