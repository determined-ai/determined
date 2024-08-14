import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import React, { useCallback, useMemo, useState } from 'react';

import ExperimentCreateModalComponent, {
  CreateExperimentType,
} from 'components/ExperimentCreateModal';
import HyperparameterSearchModalComponent from 'components/HyperparameterSearchModal';
import PageHeaderFoldable, { renderOptionLabel } from 'components/PageHeaderFoldable';
import { UNMANAGED_MESSAGE } from 'constant';
import { terminalRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import { ActionOptions } from 'pages/ExperimentDetails/ExperimentDetailsHeader';
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

// prettier-ignore
const TrialDetailsHeader: React.FC<Props> = ({ experiment, fetchTrialDetails, trial }: Props) => {
  const [isRunningTensorBoard, setIsRunningTensorBoard] = useState<boolean>(false);
  const [trialNeverData, setTrialNeverData] = useState<boolean>(false);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const handleModalClose = useCallback(() => fetchTrialDetails(), [fetchTrialDetails]);

  const ExperimentCreateModal = useModal(ExperimentCreateModalComponent);
  const HyperparameterSearchModal = useModal(HyperparameterSearchModalComponent);

  const handleHyperparameterSearch = useCallback(() => {
    HyperparameterSearchModal.open();
  }, [HyperparameterSearchModal]);

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

  const headerOptions = useMemo<ActionOptions[]>(() => {
    const options: ActionOptions[] = [];

    if (!trialNeverData) {
      options.push({
        key: Action.OpenTensorBoard,
        menuOptions: [
          {
            icon: <Icon decorative name="tensor-board" size="small" />,
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
          },
        ],
      });
    }

    if (canActionExperiment(ExperimentAction.ContinueTrial, experiment, trial)) {
      const menuOption: ActionOptions['menuOptions'][number] = {
        disabled: experiment.unmanaged,
        icon: <Icon decorative name="fork" size="small" />,
        key: Action.ContinueTrial,
        label: `Continue ${f_flat_runs ? 'Run' : 'Trial'}`,
      };
      if (!trial.bestAvailableCheckpoint) {
        menuOption.tooltip = `No checkpoints found. Cannot continue ${f_flat_runs ? 'run' : 'trial'}`;
      } else {
        menuOption.onClick = ExperimentCreateModal.open,
        experiment.unmanaged && (menuOption.tooltip = UNMANAGED_MESSAGE);
      }
      options.push({
        key: Action.ContinueTrial,
        menuOptions: [ menuOption ],
      });
    }

    if (
      canActionExperiment(ExperimentAction.HyperparameterSearch, experiment, trial) &&
      !experiment.unmanaged
    ) {
      options.push({
        key: Action.HyperparameterSearch,
        menuOptions: [
          {
            key: Action.HyperparameterSearch,
            label: 'Hyperparameter Search',
            onClick: handleHyperparameterSearch,
          },
        ],
      });
    }

    return options;
  }, [
    experiment,
    f_flat_runs,
    fetchTrialDetails,
    ExperimentCreateModal,
    handleHyperparameterSearch,
    isRunningTensorBoard,
    trial,
    trialNeverData,
  ]);

  return (
    <>
      <PageHeaderFoldable
        leftContent={<TrialHeaderLeft experiment={experiment} trial={trial} />}
        options={headerOptions.map((option) => ({
          content: option?.content
            ? option.content
            : option.menuOptions.map((menuOption) => (
              <Button
                disabled={menuOption.disabled || !menuOption.onClick}
                icon={menuOption?.icon}
                key={menuOption.key}
                loading={menuOption.isLoading}
                onClick={menuOption.onClick}>
                {renderOptionLabel(menuOption)}
              </Button>
            )),
          key: option.key,
          menuOptions: option.menuOptions,
        }))}
      />
      <ExperimentCreateModal.Component
        experiment={experiment}
        trial={trial}
        type={CreateExperimentType.ContinueTrial}
        onClose={handleModalClose}
      />
      <HyperparameterSearchModal.Component
        closeModal={HyperparameterSearchModal.close}
        experiment={experiment}
        trial={trial}
      />
    </>
  );
};

export default TrialDetailsHeader;
