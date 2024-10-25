import { Modal } from 'hew/Modal';

import { UNMANAGED_EXPERIMENT_ANNOTATION_MESSAGE } from 'constant';
import { openOrCreateTensorBoardSearches } from 'services/api';
import { ProjectExperiment } from 'types';
import handleError from 'utils/error';
import { openCommandResponse } from 'utils/wait';

interface Props {
  selectedSearches: ProjectExperiment[];
  workspaceId?: number;
}

const SearchTensorBoardModal = ({ workspaceId, selectedSearches }: Props): JSX.Element => {
  const handleSubmit = async () => {
    const managedSearchIds = selectedSearches.filter((exp) => !exp.unmanaged).map((exp) => exp.id);
    openCommandResponse(
      await openOrCreateTensorBoardSearches({
        searchIds: managedSearchIds,
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

export default SearchTensorBoardModal;
