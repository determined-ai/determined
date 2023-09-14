import { Modal } from 'components/kit/Modal';
import { UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE } from 'constant';
import { openOrCreateTensorBoard } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import { ProjectExperiment } from 'types';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

interface Props {
  selectedExperiments: ProjectExperiment[];
  filters?: V1BulkExperimentFilters;
  workspaceId?: number;
}

const ExperimentTensorBoardModal = ({
  workspaceId,
  selectedExperiments,
  filters,
}: Props): JSX.Element => {
  const handleSubmit = async () => {
    const managedExperimentIds = selectedExperiments
      .filter((exp) => !exp.unmanaged)
      .map((exp) => exp.id);
    openCommandResponse(
      await openOrCreateTensorBoard({ experimentIds: managedExperimentIds, filters, workspaceId }),
    );
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'confirm',
      }}
      title="Tensorboard confirmation">
      {UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE}
    </Modal>
  );
};

export default ExperimentTensorBoardModal;
