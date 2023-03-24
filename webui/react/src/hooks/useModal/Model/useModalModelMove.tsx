import { ModalFuncProps } from 'antd/es/modal/Modal';
import { LabeledValue } from 'antd/es/select';
import { useCallback } from 'react';

import Form from 'components/kit/Form';
import Select from 'components/kit/Select';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { WorkspaceDetailsTab } from 'pages/WorkspaceDetails';
import { paths } from 'routes/utils';
import { moveModel } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { useWorkspaces } from 'stores/workspaces';
import { ModelItem } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

type FormInputs = {
  workspaceId: number;
};

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (model: ModelItem) => void;
}

const useModalModelMove = ({ onClose }: Props = {}): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const { canMoveModel } = usePermissions();
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });
  const workspaces = Loadable.getOrElse([], useWorkspaces());

  const getModalProps = useCallback(
    (model: ModelItem): ModalFuncProps => {
      const handleOk = async () => {
        const values = await form.validateFields();
        try {
          await moveModel({ destinationWorkspaceId: values.workspaceId, modelName: model.name });
          const workspaceName = workspaces.find((ws) => ws.id === values.workspaceId)?.name;
          const path = paths.workspaceDetails(
            values.workspaceId,
            WorkspaceDetailsTab.ModelRegistry,
          );
          notification.success({
            description: (
              <div>
                <p>
                  {model.name} moved to workspace {workspaceName}
                </p>
                <Link path={path}>View Workspace</Link>
              </div>
            ),
            key: 'move-model-notification',
            message: 'Successfully Moved',
          });
        } catch (e) {
          handleError(e, { publicSubject: `Unable to move model ${model.id}.`, silent: false });
        } finally {
          form.resetFields();
        }
      };

      const handleClose = () => {
        form.resetFields();
        onClose?.();
      };

      return {
        closable: true,
        content: (
          <Form autoComplete="off" form={form} layout="vertical">
            <Form.Item
              label="Workspace"
              name="workspaceId"
              rules={[{ message: 'Please select a workspace', required: true }]}>
              <Select
                filterOption={(input, option) =>
                  (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
                }
                filterSort={(a: LabeledValue, b: LabeledValue) =>
                  (a?.label ?? '') < (b?.label ?? '') ? 1 : -1
                }
                options={workspaces
                  .filter(
                    (ws) =>
                      ws.id !== model.workspaceId && canMoveModel({ destination: { id: ws.id } }),
                  )
                  .map((ws) => ({ label: ws.name, value: ws.id }))}
                placeholder="Select a workspace"
              />
            </Form.Item>
          </Form>
        ),
        icon: null,
        okButtonProps: { type: 'primary' },
        okText: 'Move',
        onCancel: handleClose,
        onOk: handleOk,
        title: `Move a Model (${model.name})`,
      };
    },
    [canMoveModel, form, onClose, workspaces],
  );

  const modalOpen = useCallback(
    (model: ModelItem) => {
      openOrUpdate(getModalProps(model));
    },
    [getModalProps, openOrUpdate],
  );

  return { modalOpen, ...modalHook };
};

export default useModalModelMove;
