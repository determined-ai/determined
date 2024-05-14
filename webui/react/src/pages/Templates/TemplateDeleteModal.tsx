import { Modal } from 'hew/Modal';

import { deleteTaskTemplate } from 'services/api';
import { Template } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

interface Props {
  template?: Template;
  onSuccess?: () => void;
}

const TemplateDeleteModalComponent: React.FC<Props> = ({ template, onSuccess }) => {
  const handleOk = async () => {
    if (!template) return;
    try {
      await deleteTaskTemplate({ name: template.name });
      onSuccess?.();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete template.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  };

  return (
    <Modal
      danger
      size="small"
      submit={{
        handleError,
        handler: handleOk,
        text: 'Delete Template',
      }}
      title="Confirm Delete Template">
      <div>Are you sure you want to delete this template &quot;{template?.name}&quot;</div>
    </Modal>
  );
};

export default TemplateDeleteModalComponent;
