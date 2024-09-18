import Alert from 'hew/Alert';
import Button from 'hew/Button';
import CodeEditor from 'hew/CodeEditor';
import Column from 'hew/Column';
import Form, { hasErrors } from 'hew/Form';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import { useState } from 'react';

import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import { getWorkspaceConfigPolicies, updateWorkspaceConfigPolicies } from 'services/api';
import handleError from 'utils/error';

interface Props {
  workspaceId?: number;
}

type FormInputs = {
  task: string;
};

const ConfigPolicies: React.FC<Props> = ({ workspaceId }: Props) => {
  const confirm = useConfirm();
  const { openToast } = useToast();
  const { canModifyWorkspaceConfigPolicies, loading: rbacLoading } = usePermissions();
  const [form] = Form.useForm<FormInputs>();

  const [disabled, setDisabled] = useState(true);

  const APPLY_MESSAGE = "You're about to apply these config policies to this workspace.";
  const VIEW_MESSAGE = 'An admin applied these config policies to this workspace.';

  const updatePolicies = async () => {
    if (workspaceId) {
      try {
        await updateWorkspaceConfigPolicies({
          configPolicies: form.getFieldValue('task'),
          workloadType: 'NTSC',
          workspaceId,
        });
        openToast({ title: 'Config policies updated' });
      } catch (error) {
        handleError(error);
      }
    }
  };

  const confirmApply = () => {
    confirm({
      content: (
        <span>
          This will impact{' '}
          <strong>
            <u>all</u>
          </strong>{' '}
          underlying projects and their experiments in this workspace.
        </span>
      ),
      okText: 'Apply',
      onConfirm: updatePolicies,
      onError: handleError,
      size: 'medium',
      title: APPLY_MESSAGE,
    });
  };

  const loadableTaskConfigPolicies: Loadable<string | undefined> = useAsync(async () => {
    if (workspaceId) {
      const response = await getWorkspaceConfigPolicies({
        workloadType: 'NTSC',
        workspaceId,
      });
      return response.configPolicies;
    }
    return NotLoaded;
  }, [workspaceId]);

  const initialTaskYAML = yaml.dump(loadableTaskConfigPolicies.getOrElse(undefined));

  const handleChange = () => {
    setDisabled(hasErrors(form) || form.getFieldValue('task') === initialTaskYAML);
  };

  if (rbacLoading) return <Spinner spinning />;

  return (
    <Column>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          {canModifyWorkspaceConfigPolicies ? (
            <Alert
              action={
                <Button disabled={disabled} onClick={confirmApply}>
                  Apply
                </Button>
              }
              message={APPLY_MESSAGE}
              showIcon
            />
          ) : (
            <Alert message={VIEW_MESSAGE} showIcon />
          )}
        </div>
      </Row>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          <Form form={form} onFieldsChange={handleChange}>
            <Form.Item
              name="task"
              rules={[
                {
                  validator: (_, value) => {
                    try {
                      yaml.load(value);
                      return Promise.resolve();
                    } catch (err: unknown) {
                      return Promise.reject(
                        new Error(
                          `Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`,
                        ),
                      );
                    }
                  },
                },
              ]}>
              <CodeEditor
                file={initialTaskYAML}
                files={[{ key: 'task', title: 'Task Config Policies' }]}
                onError={(error) => {
                  handleError(error);
                }}
              />
            </Form.Item>
          </Form>
        </div>
      </Row>
    </Column>
  );
};

export default ConfigPolicies;
