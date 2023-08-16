import React from 'react';

import { Modal } from 'components/kit/Modal';
import { paths } from 'routes/utils';
import { deleteExperiment } from 'services/api';
import { ExperimentBase } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

export const BUTTON_TEXT = 'Delete';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentDeleteModalComponent: React.FC<Props> = ({ experiment }: Props) => {
  const handleSubmit = async () => {
    try {
      await deleteExperiment({ experimentId: experiment.id });
      routeToReactUrl(paths.projectDetails(experiment.projectId));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: BUTTON_TEXT,
      }}
      title="Confirm Experiment Deletion">
      Are you sure you want to delete experiment {experiment.id}?
    </Modal>
  );
};

export default ExperimentDeleteModalComponent;
