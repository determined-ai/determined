import React, { useCallback, useMemo, useState } from 'react';

import BreadcrumbBar from 'components/BreadcrumbBar';
import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import { terminalRunStates } from 'constants/states';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import TrialHeaderLeft from 'pages/TrialDetails/Header/TrialHeaderLeft';
import { getTrialWorkloads, openOrCreateTensorBoard } from 'services/api';
import {
  ExperimentAction as Action,
  ExperimentAction,
  ExperimentBase,
  TrialDetails,
  TrialWorkloadFilter,
} from 'types';
import { canActionExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

interface Props {
  experiment: ExperimentBase;
  fetchTrialDetails: () => void;
  trial: TrialDetails;
}

const TrialDetailsHeader: React.FC<Props> = ({ experiment, fetchTrialDetails, trial }: Props) => {
  const [isRunningTensorBoard, setIsRunningTensorBoard] = useState<boolean>(false);
  const [trialNeverData, setTrialNeverData] = useState<boolean>(false);

  const handleModalClose = useCallback(() => fetchTrialDetails(), [fetchTrialDetails]);

  const ExperimentCreateModal = useModal(ExperimentCreateModalComponent);

  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment, trial });

  const handleHyperparameterSearch = useCallback(() => {
    openModalHyperparameterSearch();
  }, [openModalHyperparameterSearch]);

  useMemo(async () => {
    if (!terminalRunStates.has(trial.state)) {
      setTrialNeverData(false);
    } else {
      const wl = await getTrialWorkloads({
        filter: TrialWorkloadFilter.All,
        id: trial.id,
        limit: 1,
      });
      setTrialNeverData(wl.workloads.length === 0);
    }
  }, [trial]);

  const headerOptions = useMemo<Option[]>(() => {
    const options: Option[] = [];

    if (!trialNeverData) {
      options.push({
        icon: <Icon name="tensor-board" size="small" />,
        isLoading: isRunningTensorBoard,
        key: Action.OpenTensorBoard,
        label: 'TensorBoard',
        onClick: async () => {
          setIsRunningTensorBoard(true);
          const commandResponse = await openOrCreateTensorBoard({
            trialIds: [trial.id],
            workspaceId: experiment.workspaceId,
          });
          openCommandResponse(commandResponse);
          await fetchTrialDetails();
          setIsRunningTensorBoard(false);
        },
      });
    }

    if (canActionExperiment(ExperimentAction.ContinueTrial, experiment, trial)) {
      if (trial.bestAvailableCheckpoint !== undefined) {
        options.push({
          icon: <Icon name="fork" size="small" />,
          key: Action.ContinueTrial,
          label: 'Continue Trial',
          onClick: ExperimentCreateModal.open,
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

    if (canActionExperiment(ExperimentAction.HyperparameterSearch, experiment, trial)) {
      options.push({
        key: Action.HyperparameterSearch,
        label: 'Hyperparameter Search',
        onClick: handleHyperparameterSearch,
      });
    }

    return options;
  }, [
    experiment,
    fetchTrialDetails,
    ExperimentCreateModal,
    handleHyperparameterSearch,
    isRunningTensorBoard,
    trial,
    trialNeverData,
  ]);

  return (
    <>
      <BreadcrumbBar experiment={experiment} id={trial.id} trial={trial} type="trial" />
      <PageHeaderFoldable
        leftContent={<TrialHeaderLeft experiment={experiment} trial={trial} />}
        options={headerOptions}
      />
      <ExperimentCreateModal.Component
        experiment={experiment}
        trial={trial}
        type={CreateExperimentType.ContinueTrial}
        onClose={handleModalClose}
      />
      {modalHyperparameterSearchContextHolder}
    </>
  );
};

export default TrialDetailsHeader;
