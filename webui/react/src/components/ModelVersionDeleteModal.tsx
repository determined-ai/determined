import { Modal } from 'components/kit/Modal';
import { paths } from 'routes/utils';
import { deleteModelVersion } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ModelVersion } from 'types';
import handleError from 'utils/error';

interface Props {
  modelVersion: ModelVersion;
}

const ModelVersionDeleteModal = ({ modelVersion }: Props): JSX.Element => {
  const handleOk = async () => {
    if (!modelVersion) return Promise.reject();

    try {
      await deleteModelVersion({
        modelName: modelVersion.model.name ?? '',
        versionNum: modelVersion.version ?? 0,
      });
      routeToReactUrl(paths.modelDetails(String(modelVersion.model.id)));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: `Unable to delete model version ${modelVersion.version}.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      danger
      size="small"
      submit={{ handler: handleOk, text: 'Delete Version' }}
      title="Confirm Delete Model Version">
      <div>
        Are you sure you want to delete &quot; Version {modelVersion.version}&quot; from this model?
      </div>
    </Modal>
  );
};

export default ModelVersionDeleteModal;
