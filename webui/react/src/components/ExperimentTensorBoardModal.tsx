import { Modal } from 'components/kit/Modal';
import { UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE } from 'constant';
import { openOrCreateTensorBoard } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

interface Props {
  selectedExperimentIds: number[];
  filters?: V1BulkExperimentFilters;
  workspaceId?: number;
}

const ExperimentTensorBoardModal = ({
  workspaceId,
  selectedExperimentIds,
  filters,
}: Props): JSX.Element => {
  const handleSubmit = async () => {
    openCommandResponse(
      await openOrCreateTensorBoard({ experimentIds: selectedExperimentIds, filters, workspaceId }),
    );
  };

  return (
    <Modal
      cancel
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'Confirm',
      }}
      title="TensorBoard confirmation">
      {UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE}
    </Modal>
  );
};

export default ExperimentTensorBoardModal;
