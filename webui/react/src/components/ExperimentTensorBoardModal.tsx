import { Modal } from 'hew/Modal';

import { FilterFormSetWithoutId } from 'components/FilterForm/components/type';
import { UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE } from 'constant';
import { openOrCreateTensorBoard } from 'services/api';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

interface Props {
  selectedExperiments: ExperimentItem[];
  filters?: FilterFormSetWithoutId;
  workspaceId?: number;
}

const ExperimentTensorBoardModal = ({
  workspaceId,
  selectedExperiments,
  filters,
}: Props): JSX.Element => {
  const handleSubmit = async () => {
    const managedExperimentIds = filters
      ? []
      : selectedExperiments.filter((exp) => !exp.unmanaged).map((exp) => exp.id);
    openCommandResponse(
      await openOrCreateTensorBoard({
        experimentIds: managedExperimentIds,
        searchFilters: filters && JSON.stringify(filters),
        workspaceId,
      }),
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
