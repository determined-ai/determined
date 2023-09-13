import { Modal } from 'components/kit/Modal';
import { openOrCreateTensorBoard } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

interface Props {
  experimentIds: number[];
  filters?: V1BulkExperimentFilters;
  workspaceId?: number;
}

const ExperimentTensorBoardModal = ({
  workspaceId,
  experimentIds,
  filters,
}: Props): JSX.Element => {
  const handleSubmit = async () => {
    openCommandResponse(await openOrCreateTensorBoard({ experimentIds, filters, workspaceId }));
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
      Unmanaged experiments selected, however, those experiments will be ignored
    </Modal>
  );
};

export default ExperimentTensorBoardModal;
