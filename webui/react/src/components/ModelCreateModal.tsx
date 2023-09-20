import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useObservable } from 'micro-observables';
import { useId, useState } from 'react';

import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import Select from 'components/kit/Select';
import { makeToast } from 'components/kit/Toast';
import Link from 'components/Link';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { postModel } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { Metadata } from 'types';
import handleError, { DetError, ErrorType } from 'utils/error';
import { Loadable } from 'utils/loadable';

import css from './ModelCreateModal.module.scss';

const FORM_ID = 'create-model-form';

type MetadataForm = { key?: string; value?: string }[];

type FormInputs = {
  modelDescription?: string;
  modelName: string;
  workspaceId: number;
  metadata?: MetadataForm;
  tags?: string[];
};

interface Props {
  // TODO: we should be able to remove `reason` from onClose props after modal migration
  onClose?: (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => void;
  workspaceId?: number;
}

const ModelCreateModal = ({ onClose, workspaceId }: Props): JSX.Element => {
  const idPrefix = useId();
  const { canCreateModelWorkspace } = usePermissions();
  const [isDetailExpanded, setIsDetailExpanded] = useState<boolean>(false);
  const loadableWorkspaces = useObservable(workspaceStore.workspaces);
  const isWorkspace = workspaceId !== undefined;
  const workspaces = Loadable.match(loadableWorkspaces, {
    Loaded: (ws) => ws.filter(({ id }) => canCreateModelWorkspace({ workspaceId: id })),
    NotLoaded: () => [],
  });
  const [form] = Form.useForm<FormInputs>();
  const disableWorkspaceModelCreation = isWorkspace
    ? !canCreateModelWorkspace({ workspaceId })
    : false;

  const onCreateModel = async () => {
    const values = await form.validateFields();
    const { modelDescription, modelName, workspaceId, metadata, tags } = values;
    const newMetadata: Metadata = {};
    for (const m of metadata ?? []) {
      if (m.key) {
        newMetadata[m.key] = m.value ?? '';
      }
    }

    try {
      const response = await postModel({
        description: modelDescription,
        labels: tags,
        metadata: newMetadata,
        name: modelName,
        workspaceId,
      });
      if (!response?.id) return;
      makeToast({
        description: `${modelName} created`,
        link: <Link path={paths.modelDetails(response.name)}>View Model</Link>,
        severity: 'Info',
        title: '',
      });
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create model.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create model.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    }
  };

  const onCloseModal = () => {
    form.resetFields();
    onClose?.();
  };

  const onOpenDetails = () => {
    setIsDetailExpanded(true);
  };

  return (
    <Modal
      size="medium"
      submit={{
        disabled: isWorkspace && disableWorkspaceModelCreation,
        form: idPrefix + FORM_ID,
        handleError,
        handler: onCreateModel,
        text: 'Create',
      }}
      title="Create a new model"
      onClose={onCloseModal}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
        <p className={css.directions}>
          Create a registered model to organize important checkpoints.
        </p>
        <Form.Item
          initialValue={workspaceId}
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Please select a workspace', required: true }]}>
          <Select
            disabled={!workspaces.length || isWorkspace}
            filterOption={(input, option) =>
              (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
            }
            options={workspaces.map((ws) => ({
              label: ws.name,
              value: ws.id,
            }))}
            placeholder="Select a workspace"
          />
        </Form.Item>
        <Form.Item
          label="Model name"
          name="modelName"
          rules={[{ message: 'Please input Model name', required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Description (optional)" name="modelDescription">
          <Input.TextArea />
        </Form.Item>
        {!isDetailExpanded && <Link onClick={onOpenDetails}>Add More Details...</Link>}
        {isDetailExpanded && (
          <>
            <div>
              <div className={css.label}>
                <label>Metadata (optional)</label>
              </div>
              <Form.List
                name="metadata"
                rules={[
                  {
                    validator: async (_, metadata?: MetadataForm) => {
                      const metadataKeys = metadata?.map((m) => m?.key ?? '') ?? [];
                      const metadataKeySet = new Set(metadataKeys);
                      if (metadataKeySet.size !== metadataKeys.length) {
                        return await Promise.reject(new Error('No dupelicate keys'));
                      }
                    },
                  },
                ]}>
                {(fields, { add, remove }, { errors }) => (
                  <>
                    {fields.map(({ key, name, ...restField }) => (
                      <div className={css.metadataRow} key={key}>
                        <Form.Item
                          {...restField}
                          initialValue=""
                          name={[name, 'key']}
                          rules={[
                            { message: 'Key is required', required: true, whitespace: true },
                          ]}>
                          <Input placeholder="Key" size="small" />
                        </Form.Item>
                        <Form.Item initialValue="" {...restField} name={[name, 'value']}>
                          <Input placeholder="Value" size="small" />
                        </Form.Item>
                        <MinusCircleOutlined onClick={() => remove(name)} />
                      </div>
                    ))}
                    <div className={css.formError}>
                      <Form.ErrorList errors={errors} />
                    </div>
                    <Form.Item>
                      <Button block icon={<PlusOutlined />} type="dashed" onClick={() => add()}>
                        Add metadata
                      </Button>
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </div>
            <div>
              <div className={css.label}>
                <label>Tags (optional)</label>
              </div>
              <Form.List
                name="tags"
                rules={[
                  {
                    validator: async (_, tags?: string[]) => {
                      const tagSet = new Set(tags);
                      if (tags && tagSet.size !== tags.length) {
                        return await Promise.reject(new Error('No dupelicate tags'));
                      }
                    },
                  },
                ]}>
                {(fields, { add, remove }, { errors }) => (
                  <>
                    <div className={css.tagContainer}>
                      {fields.map(({ key, name, ...restField }) => (
                        <div className={css.tagRow} key={key}>
                          <Form.Item
                            {...restField}
                            initialValue=""
                            name={name}
                            rules={[
                              { message: 'Tag is required', required: true, whitespace: true },
                            ]}>
                            <Input placeholder="Tag" size="small" type="text" />
                          </Form.Item>
                          <MinusCircleOutlined onClick={() => remove(name)} />
                        </div>
                      ))}
                    </div>
                    <div className={css.formError}>
                      <Form.ErrorList errors={errors} />
                    </div>
                    <Form.Item>
                      <Button block icon={<PlusOutlined />} type="dashed" onClick={() => add()}>
                        Add tag
                      </Button>
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </div>
          </>
        )}
      </Form>
    </Modal>
  );
};

export default ModelCreateModal;
