import Form from 'hew/Form';
import { Modal } from 'hew/Modal';
import Select from 'hew/Select';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import { useId } from 'react';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { WorkspaceDetailsTab } from 'pages/WorkspaceDetails';
import { paths } from 'routes/utils';
import { moveModel } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ModelItem } from 'types';
import handleError from 'utils/error';

const FORM_ID = 'move-model-form';

type FormInputs = {
  workspaceId: number;
};

interface Props {
  model: ModelItem;
}

const ModelMoveModal = ({ model }: Props): JSX.Element => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const { canMoveModel } = usePermissions();
  const { openToast } = useToast();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const handleOk = async () => {
    const values = await form.validateFields();
    try {
      await moveModel({ destinationWorkspaceId: values.workspaceId, modelName: model.name });
      const workspaceName = workspaces.find((ws) => ws.id === values.workspaceId)?.name;
      const path =
        values.workspaceId === 1
          ? paths.modelList()
          : paths.workspaceDetails(values.workspaceId, WorkspaceDetailsTab.ModelRegistry);
      openToast({
        description: `${model.name} moved to workspace ${workspaceName}`,
        link: <Link path={path}>View Workspace</Link>,
        severity: 'Confirm',
        title: 'Successfully Moved',
      });
    } catch (e) {
      handleError(e, { publicSubject: `Unable to move model ${model.id}.`, silent: false });
    } finally {
      form.resetFields();
    }
  };

  const handleClose = () => {
    form.resetFields();
  };

  return (
    <Modal
      size="small"
      submit={{ form: idPrefix + FORM_ID, handleError, handler: handleOk, text: 'Move' }}
      title={`Move a Model (${model.name})`}
      onClose={handleClose}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
        <Form.Item
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Please select a workspace', required: true }]}>
          <Select
            filterOption={(input, option) =>
              (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
            }
            filterSort={(a, b) => ((a?.label ?? '') < (b?.label ?? '') ? -1 : 1)}
            options={workspaces
              .filter(
                (ws) => ws.id !== model.workspaceId && canMoveModel({ destination: { id: ws.id } }),
              )
              .map((ws) => ({ label: ws.name, value: ws.id }))}
            placeholder="Select a workspace"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ModelMoveModal;
