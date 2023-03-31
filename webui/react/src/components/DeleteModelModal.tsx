import { Modal } from 'components/kit/Modal';
import { paths } from 'routes/utils';
import { deleteModel } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { ModelItem } from 'types';
import handleError from 'utils/error';

interface Props {
  model: ModelItem;
}

const DeleteModelModal = ({ model }: Props): JSX.Element => {
  const handleOk = async () => {
    try {
      await deleteModel({ modelName: model.name });
      routeToReactUrl(paths.modelList());
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete model.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      danger
      size="small"
      submit={{ handler: handleOk, text: 'Delete Model' }}
      title="Confirm Delete Model">
      <div>
        Are you sure you want to delete this model &quot;{model?.name}&quot; and all of its versions
        from the model registry?
      </div>
    </Modal>
  );
};

export default DeleteModelModal;
